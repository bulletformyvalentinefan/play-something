package com.spotify.group.domain.track.dto;

import java.util.List;

public record DeezerSearchResponse(
        List<DeezerTrackResponse> data,
        Integer total
) {}