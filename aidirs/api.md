# API Configuration for Bits (Cheers)

## Feature
When bits (cheers) are received in chat, send a notification to a configurable API endpoint.

## Data Payload
The API will receive a POST request with the following JSON payload:

```json
{
  "name": "username",
  "content": "message text",
  "name_color": "#hexcolor"
}
```

## Configuration

### Config Structure
Add a new `Bits` section to the TOML config:

```toml
[twitch]
channel = ""
user = ""
oauth = ""
refresh = ""
refresh_api = ""

[twitch.bits]
enabled = false
endpoint = ""
```

### Fields
- `enabled` (bool): Whether to send API calls for bits events (default: false)
- `endpoint` (string): The HTTP URL to POST to when bits are received

## Implementation Details

### 1. Config Changes (`internal/config/config.go`)
- Add `Bits` struct with `Enabled` and `Endpoint` fields
- Update `Twitch` struct to include `Bits Bits`
- Add defaults in `defaultTwitch()` (enabled: false, endpoint: "")

### 2. Twitch Service Changes (`internal/twitch/twitch.go`)
- Store bits config in `Service` struct
- When message with `Bits > 0` is received and bits API is enabled:
  - Make async (non-blocking) HTTP POST request
  - Send JSON payload with name, content, and name_color
  - Silently ignore errors (don't block chat)

### 3. Integration
- Update `New()` to pass bits config to service
- Fire API call asynchronously only when:
  - `bits.Enabled` is true
  - `bits.Endpoint` is not empty
  - Message has bits (`msg.Bits > 0`)

## Open Questions (for implementation)
1. **HTTP Method**: POST with JSON body
2. **Bits Amount**: Consider sending bits count as optional field?
3. **Error Handling**: Silently ignore failures to not block chat flow
4. **Auth**: No auth headers for now (can be added later if needed)
