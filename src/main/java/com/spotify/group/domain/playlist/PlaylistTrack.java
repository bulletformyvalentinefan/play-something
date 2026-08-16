package com.spotify.group.domain.playlist;

import jakarta.persistence.*;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;

import java.time.LocalDateTime;
import java.util.UUID;

@Entity
@Table(name = "playlist_tracks")
@Getter
@Setter
@NoArgsConstructor
public class PlaylistTrack {

    @Id
    @GeneratedValue(strategy = GenerationType.UUID)
    private UUID id;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "playlist_id", nullable = false)
    private Playlist playlist;

    @Column(name = "deezer_track_id", nullable = false)
    private Long deezerTrackId;

    @Column(name = "added_at")
    private LocalDateTime addedAt = LocalDateTime.now();

    public PlaylistTrack(Playlist playlist, Long deezerTrackId) {
        this.playlist = playlist;
        this.deezerTrackId = deezerTrackId;
        this.addedAt = LocalDateTime.now();
    }
}