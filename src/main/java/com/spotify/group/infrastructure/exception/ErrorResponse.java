package com.spotify.group.infrastructure.exception;

public record ErrorResponse(
        int status,
        String message
) {
}