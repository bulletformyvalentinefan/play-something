package com.spotify.group.domain.playlist;

import com.spotify.group.domain.user.User;
import com.spotify.group.domain.user.UserRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;
import java.util.UUID;

@Service
@RequiredArgsConstructor
public class PlaylistService {

    private final PlaylistRepository playlistRepository;
    private final PlaylistTrackRepository playlistTrackRepository;
    private final UserRepository userRepository;

    @Transactional
    public PlaylistResponse createPlaylist(CreatePlaylistRequest request) {
        User user = userRepository.findById(request.userId())
                .orElseThrow(() -> new IllegalArgumentException("Usuario no encontrado con ID: " + request.userId()));

        Playlist playlist = new Playlist();
        playlist.setTitulo(request.titulo());
        playlist.setDescripcion(request.descripcion());
        playlist.setEsPublica(request.esPublica() != null ? request.esPublica() : true);
        playlist.setUser(user);

        return PlaylistResponse.fromEntity(playlistRepository.save(playlist));
    }

    @Transactional(readOnly = true)
    public List<PlaylistResponse> getPlaylistsByUserId(UUID userId) {
        return playlistRepository.findByUserId(userId).stream()
                .map(PlaylistResponse::fromEntity)
                .toList();
    }

    @Transactional(readOnly = true)
    public PlaylistResponse getPlaylistById(UUID playlistId) {
        Playlist playlist = playlistRepository.findById(playlistId)
                .orElseThrow(() -> new IllegalArgumentException("Playlist no encontrada con ID: " + playlistId));
        return PlaylistResponse.fromEntity(playlist);
    }

    @Transactional
    public void addTrackToPlaylist(UUID playlistId, Long deezerTrackId) {
        Playlist playlist = playlistRepository.findById(playlistId)
                .orElseThrow(() -> new IllegalArgumentException("Playlist no encontrada con ID: " + playlistId));

        playlistTrackRepository.findByPlaylistIdAndDeezerTrackId(playlistId, deezerTrackId)
                .ifPresent(track -> {
                    throw new IllegalArgumentException("La canción ya se encuentra en la playlist");
                });

        PlaylistTrack playlistTrack = new PlaylistTrack(playlist, deezerTrackId);
        playlistTrackRepository.save(playlistTrack);
    }

    @Transactional
    public void removeTrackFromPlaylist(UUID playlistId, Long deezerTrackId) {
        playlistTrackRepository.deleteByPlaylistIdAndDeezerTrackId(playlistId, deezerTrackId);
    }
}