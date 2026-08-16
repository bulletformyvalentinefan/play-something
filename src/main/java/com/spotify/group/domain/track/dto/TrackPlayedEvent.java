package com.spotify.group.domain.track.dto;

import java.time.LocalDateTime;
import java.util.UUID;

public record TrackPlayedEvent(
        UUID userId,
        Long trackId,
        LocalDateTime playedAt
) {}