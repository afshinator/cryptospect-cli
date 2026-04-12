# Agent Guide: cryptospect-cli

## What This Tool Does
Computes crypto market regime metrics for agentic consumption. Outputs machine-readable JSON optimized for low token count.

## CLI Signatures

    cryptospect-cli regime        --asset <SYM> --window <DURATION> --output json
    cryptospect-cli zscore        --asset <SYM> --period <DURATION> --output json
    cryptospect-cli rvol          --asset <SYM> --output json
    cryptospect-cli correlation   --pair <SYM,SYM> --window <DURATION> --output json
    cryptospect-cli summary       --assets <SYM,SYM,...> --output json|nl

## Orchestration Playbook
- ALWAYS run `regime` before other commands to establish market context
- Use `zscore` + `rvol` together for confirmation signals
- Only run `correlation` when cross-asset comparison is needed
- Chain: regime → zscore → rvol → (optional) correlation → summary

## Output Envelope
Every invocation returns this structure on stdout:

    {
      "status": "ok|error",
      "data": {},
      "error": {
        "code": 429,
        "msg": "rate_limited",
        "retry_after_sec": 60,
        "source": "coingecko"
      }
    }

## Error Handling
- On rate limit (429): parse retry_after_sec, sleep, retry
- On error: check error.source to determine which API failed
- Never parse stderr — it contains debug logs only

## API Key Injection
Precedence: --api-key flag > CRYPTOSPECT_API_KEY env var > ~/.cryptospect.yaml