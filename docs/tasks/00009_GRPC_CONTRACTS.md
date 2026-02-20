---
id: 9
title: Add gRPC contract verification support
status: draft
created: 2026-02-19
tags: [maslow, verification, contract, grpc]
---

## Objective

Add a `grpc` action type to contracts so Maslow can verify gRPC services, not just HTTP APIs.

## Requirements

- New step action type `grpc` with fields: service, method, proto_path, message (JSON), metadata
- Expectation support: status code (gRPC codes), response json_path, body_contains, body_matches
- Use grpcurl or a native Go gRPC client for execution
- Streaming support (at minimum server-streaming with message count assertions)

## Notes

- Many microservice architectures use gRPC internally even if HTTP is the public API
- Proto file discovery could leverage refs in maslow.yaml
- Consider protobuf JSON encoding for message payloads
