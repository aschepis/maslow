---
id: 12
title: Add WebSocket contract verification support
status: draft
created: 2026-02-19
tags: [maslow, verification, contract, websocket]
---

## Objective

Add a `websocket` action type to contracts so Maslow can verify real-time WebSocket-based services.

## Requirements

- New step action type `websocket` with fields: url, send (message to send), timeout
- Expectation support: message_contains, message_matches, json_path on received messages, message_count
- Support connect, send, receive, and close lifecycle
- Handle binary and text message frames

## Notes

- Important for chat apps, real-time dashboards, game servers, and event-driven architectures
- Consider a sequence-based approach: connect, send N messages, expect M responses
- Timeout handling is critical for receive operations
