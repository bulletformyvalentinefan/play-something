package com.spotify.group.domain.playlist;

import org.springframework.data.jpa.repository.JpaRepository;
import java.util.Optional;
import java.util.UUID;

public interface PlaylistTrackRepository extends JpaRepository<PlaylistTrack, UUID> {

    Optional<PlaylistTrack> findByPlaylistIdAndDeezerTrackId(UUID playlistId, Long deezerTrackId);
    void deleteByPlaylistIdAndDeezerTrackId(UUID playlistId, Long deezerTrackId);
}