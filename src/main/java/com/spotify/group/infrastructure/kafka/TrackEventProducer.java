package com.spotify.group.infrastructure.kafka;

import com.spotify.group.domain.track.dto.TrackPlayedEvent;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.stereotype.Service;

import java.time.LocalDateTime;
import java.util.UUID;

@Service
@RequiredArgsConstructor
@Slf4j
public class TrackEventProducer {

    private static final String TOPIC = "track-played-topic";
    private final KafkaTemplate<String, Object> kafkaTemplate;

    public void emitTrackPlayed(UUID userId, Long trackId) {
        TrackPlayedEvent event = new TrackPlayedEvent(userId, trackId, LocalDateTime.now());
        kafkaTemplate.send(TOPIC, trackId.toString(), event);
        log.info("Evento emitido a Kafka en tópico [{}]: Track {} reproducido por {}", TOPIC, trackId, userId);
    }
}