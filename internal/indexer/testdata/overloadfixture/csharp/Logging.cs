namespace Logging;

public class LogEvent
{
}

public interface ILogger
{
    void Write(LogEvent evt);

    void Write(string message);
}

public class Processor
{
#if FEATURE_SPAN
    public void Process(string template)
    {
    }
#else
    public void Process(string template)
    {
    }
#endif

    public void Run(string s)
    {
        Process(s);
    }
}

public class Emitter
{
    public void Emit(ILogger log, LogEvent evt)
    {
        log.Write(evt);
    }
}
