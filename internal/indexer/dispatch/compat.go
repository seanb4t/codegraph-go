package dispatch

import "strings"

// Compatible reports whether an argument of static type arg can be passed
// to a parameter of type param in language lang, as far as the names
// tell (design D3 of the index-invariants change). Both types are the
// normalized names the extractors record (goextract.TypeName). The table
// is lenient on purpose: an unknown argument type matches anything, and a
// widening or boxing conversion that the language allows matches. A
// lenient match keeps a candidate, so the worst outcome is an ambiguous
// call, never a wrong one.
func Compatible(lang, arg, param string) bool {
	arg, param = normalize(arg), normalize(param)
	if arg == "" || param == "" || arg == param {
		return true
	}
	if isTypeVariable(param) || topTypes[param] {
		return true
	}
	if a, p := eraseGenerics(arg), eraseGenerics(param); a == p {
		return true
	}
	if wide, ok := widening[lang]; ok {
		if targets, ok := wide[arg]; ok && targets[param] {
			return true
		}
	}
	if box, ok := boxing[lang]; ok {
		if box[arg] == param || box[param] == arg {
			return true
		}
	}
	return false
}

func normalize(t string) string {
	t = strings.TrimSpace(t)
	for _, p := range []string{"final ", "const ", "readonly ", "in ", "out ", "ref "} {
		t = strings.TrimPrefix(t, p)
	}
	return strings.Join(strings.Fields(t), " ")
}

// eraseGenerics drops type arguments: List<String> and List<T> compare as
// List.
func eraseGenerics(t string) string {
	if i := strings.IndexAny(t, "<["); i > 0 && !strings.HasSuffix(t, "[]") {
		return t[:i]
	}
	return t
}

// isTypeVariable reports whether t looks like a type parameter: one or
// two capital letters, with an optional digit, such as T, K, V, T1 or TK.
func isTypeVariable(t string) bool {
	if len(t) == 0 || len(t) > 3 {
		return false
	}
	for i, r := range t {
		switch {
		case r >= 'A' && r <= 'Z':
		case i > 0 && r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}

// Final reports whether t is a type of lang that no type in the index
// can extend or implement: a primitive, a boxed primitive, or a sealed
// core type such as Java's String. A comparison between a final type and
// a type the index holds can reject a candidate. A comparison with a
// library interface or class that an indexed type can implement, such as
// Iterable or Object, cannot, because the supertype chain leaves the
// index.
func Final(lang, t string) bool {
	t = normalize(t)
	return finals[lang][eraseGenerics(t)]
}

var finals = map[string]map[string]bool{
	"java": set("void", "int", "long", "double", "float", "boolean", "char", "byte", "short",
		"String", "Integer", "Long", "Double", "Float", "Boolean", "Character", "Byte", "Short"),
	"csharp": set("void", "int", "long", "double", "float", "bool", "char", "byte", "short", "decimal",
		"uint", "ulong", "ushort", "sbyte", "string", "String", "Int32", "Int64", "Double", "Single",
		"Boolean", "Char", "Decimal"),
	"kotlin": set("Int", "Long", "Double", "Float", "Boolean", "Char", "Byte", "Short", "String", "Unit", "Nothing"),
	"swift":  set("Int", "UInt", "Double", "Float", "Bool", "String", "Character", "Void"),
	"cpp":    set("void", "int", "long", "short", "char", "float", "double", "bool", "unsigned", "size_t"),
	"c":      set("void", "int", "long", "short", "char", "float", "double", "unsigned", "size_t"),
	"go": set("int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "string", "bool", "byte", "rune", "uintptr", "complex64", "complex128"),
	"typescript": set("string", "number", "boolean", "void", "null", "undefined", "never", "bigint", "symbol"),
	"tsx":        set("string", "number", "boolean", "void", "null", "undefined", "never", "bigint", "symbol"),
	"javascript": set("string", "number", "boolean", "void", "null", "undefined"),
	"python":     set("str", "int", "float", "bool", "bytes", "None"),
	"rust": set("i8", "i16", "i32", "i64", "i128", "isize", "u8", "u16", "u32", "u64", "u128", "usize",
		"f32", "f64", "bool", "char", "str", "String"),
	"php":  set("string", "int", "float", "bool", "null", "void"),
	"ruby": set("String", "Integer", "Float", "Symbol", "NilClass", "TrueClass", "FalseClass"),
}

// Builtin reports whether t is a type the language defines, such as a
// primitive or a core library type. The resolver compares an argument
// type with a parameter type only when it knows both: a builtin, or a
// type the index holds. A type it knows neither way is foreign, and a
// foreign type never rejects a candidate.
func Builtin(lang, t string) bool {
	t = normalize(t)
	if topTypes[t] || isTypeVariable(t) {
		return true
	}
	return builtins[lang][eraseGenerics(t)]
}

var builtins = map[string]map[string]bool{
	"java": set("void", "int", "long", "double", "float", "boolean", "char", "byte", "short",
		"String", "Integer", "Long", "Double", "Float", "Boolean", "Character", "Byte", "Short",
		"Number", "CharSequence", "List", "Map", "Set", "Collection", "Iterable", "Optional"),
	"csharp": set("void", "int", "long", "double", "float", "bool", "char", "byte", "short", "decimal",
		"uint", "ulong", "ushort", "sbyte", "string", "String", "Int32", "Int64", "Double", "Single",
		"Boolean", "Char", "Decimal", "List", "Dictionary", "IEnumerable", "Task", "Span", "ReadOnlySpan"),
	"kotlin": set("Int", "Long", "Double", "Float", "Boolean", "Char", "Byte", "Short", "String",
		"Unit", "Nothing", "List", "Map", "Set", "Array"),
	"swift": set("Int", "UInt", "Double", "Float", "Bool", "String", "Character", "Void",
		"AnyObject", "Array", "Dictionary", "Set", "Optional"),
	"cpp": set("void", "int", "long", "short", "char", "float", "double", "bool", "unsigned",
		"size_t", "string", "std::string", "std::vector", "vector"),
	"c": set("void", "int", "long", "short", "char", "float", "double", "unsigned", "size_t"),
	"go": set("int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "string", "bool", "byte", "rune", "error", "uintptr", "complex64", "complex128"),
	"typescript": set("string", "number", "boolean", "void", "null", "undefined", "never", "bigint", "symbol",
		"Array", "Promise", "Record", "Map", "Set"),
	"tsx":        set("string", "number", "boolean", "void", "null", "undefined", "never", "bigint", "symbol"),
	"javascript": set("string", "number", "boolean", "void", "null", "undefined"),
	"python":     set("str", "int", "float", "bool", "bytes", "list", "dict", "set", "tuple", "None"),
	"rust": set("i8", "i16", "i32", "i64", "i128", "isize", "u8", "u16", "u32", "u64", "u128", "usize",
		"f32", "f64", "bool", "char", "str", "String", "Vec", "Option", "Result", "Box"),
	"php":  set("string", "int", "float", "bool", "array", "object", "null", "void", "iterable", "callable"),
	"ruby": set("String", "Integer", "Float", "Symbol", "Array", "Hash", "NilClass", "TrueClass", "FalseClass"),
}

// topTypes accept any argument.
var topTypes = map[string]bool{
	"Object": true, "object": true, "any": true, "unknown": true, "dynamic": true,
	"Any": true, "interface{}": true, "mixed": true, "void*": true, "auto": true,
}

func set(names ...string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}

// widening lists, per language, the parameter types an argument type can
// widen to without a cast. Kotlin and Swift allow a literal, not a
// variable, to change its numeric type. The table cannot tell the two
// apart, so it allows both, which can leave a call ambiguous but never
// resolves it to the wrong overload.
var widening = map[string]map[string]map[string]bool{
	"java": {
		"byte":  set("short", "int", "long", "float", "double"),
		"short": set("int", "long", "float", "double"),
		"char":  set("int", "long", "float", "double"),
		"int":   set("long", "float", "double"),
		"long":  set("float", "double"),
		"float": set("double"),
	},
	"csharp": {
		"sbyte":  set("short", "int", "long", "float", "double", "decimal"),
		"byte":   set("short", "ushort", "int", "uint", "long", "ulong", "float", "double", "decimal"),
		"short":  set("int", "long", "float", "double", "decimal"),
		"ushort": set("int", "uint", "long", "ulong", "float", "double", "decimal"),
		"char":   set("ushort", "int", "uint", "long", "ulong", "float", "double", "decimal"),
		"int":    set("long", "float", "double", "decimal"),
		"uint":   set("long", "ulong", "float", "double", "decimal"),
		"long":   set("float", "double", "decimal"),
		"ulong":  set("float", "double", "decimal"),
		"float":  set("double"),
	},
	"kotlin": {
		"Int":   set("Long", "Float", "Double"),
		"Long":  set("Float", "Double"),
		"Float": set("Double"),
	},
	"swift": {
		"Int": set("Double", "Float"),
	},
	"cpp": {
		"char":  set("int", "long", "float", "double"),
		"int":   set("long", "float", "double"),
		"long":  set("float", "double"),
		"float": set("double"),
	},
	"c": {
		"char":  set("int", "long", "float", "double"),
		"int":   set("long", "float", "double"),
		"long":  set("float", "double"),
		"float": set("double"),
	},
}

// boxing maps a primitive to its wrapper, per language, in both directions.
var boxing = map[string]map[string]string{
	"java": {
		"int": "Integer", "long": "Long", "double": "Double", "float": "Float",
		"boolean": "Boolean", "char": "Character", "byte": "Byte", "short": "Short",
	},
	"csharp": {
		"int": "Int32", "long": "Int64", "double": "Double", "float": "Single",
		"bool": "Boolean", "char": "Char", "string": "String", "object": "Object",
	},
}
