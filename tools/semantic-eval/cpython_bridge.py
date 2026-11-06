"""Print a Python source tree's declarations and lexical bindings as JSON.

Usage: cpython_bridge.py ROOT SOURCE...

Positions are 1-based lines and 0-based UTF-8 byte columns. Locals inside a
function are omitted. Receiver attributes are `self.x` assignments inside a
method of a class. The bridge reports no dynamic call target.
"""

import ast
import json
import os
import symtable
import sys


def identifier_col(source_line: bytes, name: str, start: int) -> int:
    offset = source_line.find(name.encode(), start)
    if offset < 0:
        raise ValueError(f"{name!r} is not on its declaration line")
    return offset


class Walker(ast.NodeVisitor):
    def __init__(self, path: str, lines: list[bytes], table: symtable.SymbolTable):
        self.path, self.lines = path, lines
        self.module = path.removesuffix(".py").replace("/", ".")
        self.owner: list[str] = []
        self.kinds: list[str] = []
        self.tables = [table]
        self.rows: list[dict] = []
        self.imports: dict[str, str] = {}

    def emit(self, line: int, col: int, symbol: str, definition: bool) -> None:
        self.rows.append(
            {"path": self.path, "line": line, "col": col, "symbol": symbol, "definition": definition}
        )

    def symbol(self, owners: list[str], name: str) -> str:
        return ":".join([self.module, ".".join([*owners, name])])

    def define(self, line: int, col: int, name: str, owners: list[str] | None = None) -> None:
        self.emit(line, col, self.symbol(self.owner if owners is None else owners, name), True)

    def child_table(self, name: str, lineno: int) -> symtable.SymbolTable:
        for child in self.tables[-1].get_children():
            if child.get_name() == name and child.get_lineno() == lineno:
                return child
        raise LookupError(f"no symbol table for {name} at line {lineno}")

    def scope(self, node, kind: str) -> None:
        line = self.lines[node.lineno - 1]
        self.define(node.lineno, identifier_col(line, node.name, node.col_offset), node.name)
        for deco in node.decorator_list:
            self.visit(deco)
        self.owner.append(node.name)
        self.kinds.append(kind)
        self.tables.append(self.child_table(node.name, node.lineno))
        for child in node.body:
            self.visit(child)
        self.tables.pop()
        self.kinds.pop()
        self.owner.pop()

    def visit_FunctionDef(self, node):
        self.scope(node, "function")

    def visit_AsyncFunctionDef(self, node):
        self.scope(node, "function")

    def visit_ClassDef(self, node):
        for base in node.bases:
            self.visit(base)
        self.scope(node, "class")

    def visit_Import(self, node):
        for alias in node.names:
            self.imports[alias.asname or alias.name.split(".")[0]] = alias.name

    def visit_ImportFrom(self, node):
        for alias in node.names:
            target = f"{node.module}:{alias.name}" if node.module else alias.name
            self.imports[alias.asname or alias.name] = target

    def in_binding_scope(self) -> bool:
        return not self.kinds or self.kinds[-1] == "class"

    def bind(self, target) -> None:
        if isinstance(target, ast.Name):
            if self.in_binding_scope():
                self.define(target.lineno, target.col_offset, target.id)
        elif isinstance(target, (ast.Tuple, ast.List)):
            for item in target.elts:
                self.bind(item)
        elif isinstance(target, ast.Starred):
            self.bind(target.value)
        elif (
            isinstance(target, ast.Attribute)
            and isinstance(target.value, ast.Name)
            and target.value.id == "self"
            and len(self.kinds) >= 2
            and self.kinds[-2:] == ["class", "function"]
        ):
            self.define(target.end_lineno, target.end_col_offset - len(target.attr.encode()), target.attr, self.owner[:-1])
        else:
            self.visit(target)

    def visit_Assign(self, node):
        self.visit(node.value)
        for target in node.targets:
            self.bind(target)

    def visit_AnnAssign(self, node):
        if node.value is not None:
            self.visit(node.value)
        self.bind(node.target)

    def visit_Name(self, node):
        if not isinstance(node.ctx, ast.Load):
            return
        table = self.tables[-1]
        try:
            sym = table.lookup(node.id)
        except KeyError:
            return
        if sym.is_local() and table.get_type() == "function":
            return
        if node.id in self.imports:
            target = self.imports[node.id]
            self.emit(node.lineno, node.col_offset, target if ":" in target else target + ":", False)
            return
        if sym.is_global() or (sym.is_local() and table.get_type() == "module"):
            self.emit(node.lineno, node.col_offset, self.symbol([], node.id), False)


def main() -> None:
    root, sources = sys.argv[1], sys.argv[2:]
    rows: list[dict] = []
    for path in sources:
        if not path.endswith(".py"):
            continue
        raw = open(os.path.join(root, path), "rb").read()
        walker = Walker(path, raw.split(b"\n"), symtable.symtable(raw.decode(), path, "exec"))
        walker.visit(ast.parse(raw, path))
        rows.extend(walker.rows)
    json.dump(rows, sys.stdout)


main()
