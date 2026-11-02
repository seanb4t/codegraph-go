package com.example;

public class App {
    private Logger logger;
    private helper h;

    public void byArity(Throwable e) {
        logger.log("x", e);
    }

    public void byType(String s) {
        Sink sink = new Sink();
        sink.add(s);
    }

    public void byVariadic() {
        logger.format("a", 1, 2, 3);
    }

    public void unknownArg() {
        Sink sink = new Sink();
        sink.add(null);
    }

    public void noFit() {
        Sink sink = new Sink();
        sink.add(1, 2);
    }

    public void throughField() {
        logger.log("x");
    }

    public void lowercaseType() {
        h.assist();
    }

    public void fromNew(String s) {
        var b = new Builder();
        b.add(s);
    }

    public void fromOtherFile(String s) {
        var b = Factory.make();
        b.add(s);
    }

    public void fromUnresolved() {
        var r = lookup();
        r.run();
    }
}
