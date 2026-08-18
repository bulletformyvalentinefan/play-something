package com.spotify.group.infrastructure.kafka;

import com.spotify.group.domain.track.dto.TrackPlayedEvent;
import lombok.extern.slf4j.Slf4j;
import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.stereotype.Component;

@Component
@Slf4j
public class TrackEventConsumer {

    @KafkaListener(topics = "track-played-topic", groupId = "spotify-group")
    public void consumeTrackPlayed(TrackPlayedEvent event) {
        log.info("Evento recibido desde Kafka -> Procesando analítica para track: {} por usuario: {} en {}", event.trackId(), event.userId(), event.playedAt());
    }
}