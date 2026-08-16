package com.spotify.group.domain.playlist;

import org.springframework.data.jpa.repository.JpaRepository;
import java.util.List;
import java.util.UUID;

public interface PlaylistRepository extends JpaRepository<Playlist, UUID> {

    List<Playlist> findByUserId(UUID userId);

    UUID id(UUID id);
}