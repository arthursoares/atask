use futures_util::Stream;
use reqwest::Client;
use std::pin::Pin;
use std::task::{Context, Poll};

// ---------------------------------------------------------------------------
// Parsed SSE event
// ---------------------------------------------------------------------------

#[derive(Debug, Clone)]
pub struct SseParsedEvent {
    pub event_type: String,
    pub data: String,
    pub id: Option<String>,
}

// ---------------------------------------------------------------------------
// SSE stream
// ---------------------------------------------------------------------------

/// Connect to the SSE endpoint and return a stream of parsed events.
pub async fn connect_sse(
    base_url: &str,
    token: &str,
    last_event_id: Option<&str>,
) -> Result<SseStream, reqwest::Error> {
    let url = format!(
        "{}/events/stream?topics=task.*,project.*,section.*",
        base_url
    );

    let mut req = Client::new().get(&url).bearer_auth(token);

    if let Some(id) = last_event_id {
        req = req.header("Last-Event-ID", id);
    }

    let resp = req.send().await?.error_for_status()?;
    let byte_stream = resp.bytes_stream();

    Ok(SseStream {
        inner: Box::pin(byte_stream),
        buffer: String::new(),
        event_type: None,
        data: None,
        id: None,
    })
}

/// A stream that parses raw SSE bytes into `SseParsedEvent`s.
pub struct SseStream {
    inner: Pin<Box<dyn Stream<Item = Result<bytes::Bytes, reqwest::Error>> + Send>>,
    buffer: String,
    event_type: Option<String>,
    data: Option<String>,
    id: Option<String>,
}

impl Stream for SseStream {
    type Item = Result<SseParsedEvent, reqwest::Error>;

    fn poll_next(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        let this = self.get_mut();

        loop {
            // Try to extract a complete event from the buffer first.
            if let Some(evt) = try_parse_event(
                &mut this.buffer,
                &mut this.event_type,
                &mut this.data,
                &mut this.id,
            ) {
                return Poll::Ready(Some(Ok(evt)));
            }

            // Need more data from the byte stream.
            match this.inner.as_mut().poll_next(cx) {
                Poll::Ready(Some(Ok(chunk))) => {
                    if let Ok(text) = std::str::from_utf8(&chunk) {
                        this.buffer.push_str(text);
                    }
                }
                Poll::Ready(Some(Err(e))) => {
                    return Poll::Ready(Some(Err(e)));
                }
                Poll::Ready(None) => {
                    // Stream ended. Try to flush any remaining event.
                    if this.data.is_some() || this.event_type.is_some() {
                        let evt = flush_event(&mut this.event_type, &mut this.data, &mut this.id);
                        if let Some(e) = evt {
                            return Poll::Ready(Some(Ok(e)));
                        }
                    }
                    return Poll::Ready(None);
                }
                Poll::Pending => {
                    return Poll::Pending;
                }
            }
        }
    }
}

// ---------------------------------------------------------------------------
// SSE line-protocol parser helpers
// ---------------------------------------------------------------------------

/// Try to consume complete lines from the buffer and emit an event on blank line.
fn try_parse_event(
    buffer: &mut String,
    event_type: &mut Option<String>,
    data: &mut Option<String>,
    id: &mut Option<String>,
) -> Option<SseParsedEvent> {
    loop {
        let newline_pos = buffer.find('\n')?;
        let line = buffer[..newline_pos].trim_end_matches('\r').to_string();
        buffer.drain(..=newline_pos);

        if line.is_empty() {
            // Blank line — emit accumulated event if we have data.
            return flush_event(event_type, data, id);
        } else if let Some(value) = line.strip_prefix("event:") {
            *event_type = Some(value.trim().to_string());
        } else if let Some(value) = line.strip_prefix("data:") {
            let value = value.trim().to_string();
            match data {
                Some(existing) => {
                    existing.push('\n');
                    existing.push_str(&value);
                }
                None => *data = Some(value),
            }
        } else if let Some(value) = line.strip_prefix("id:") {
            *id = Some(value.trim().to_string());
        }
        // Ignore comment lines (starting with ':') and unknown fields.
    }
}

fn flush_event(
    event_type: &mut Option<String>,
    data: &mut Option<String>,
    id: &mut Option<String>,
) -> Option<SseParsedEvent> {
    let d = data.take()?;
    let evt = SseParsedEvent {
        event_type: event_type.take().unwrap_or_else(|| "message".to_string()),
        data: d,
        id: id.take(),
    };
    Some(evt)
}
