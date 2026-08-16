package com.spotify.group.infrastructure.kafka;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.apache.kafka.common.serialization.StringDeserializer;
import org.apache.kafka.common.serialization.StringSerializer;
import org.springframework.boot.autoconfigure.kafka.KafkaProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.kafka.config.ConcurrentKafkaListenerContainerFactory;
import org.springframework.kafka.core.ConsumerFactory;
import org.springframework.kafka.core.DefaultKafkaConsumerFactory;
import org.springframework.kafka.core.DefaultKafkaProducerFactory;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.kafka.core.ProducerFactory;
import org.springframework.kafka.support.serializer.JsonDeserializer;
import org.springframework.kafka.support.serializer.JsonSerializer;

import java.util.Map;

@Configuration
public class KafkaConfig {

    private static final String TRUSTED_PACKAGE = "com.spotify.group.domain.track.dto";

    @Bean
    public ProducerFactory<String, Object> producerFactory(KafkaProperties properties, ObjectMapper objectMapper) {
        Map<String, Object> producerProps = properties.buildProducerProperties();
        return new DefaultKafkaProducerFactory<>(producerProps, new StringSerializer(), new JsonSerializer<>(objectMapper));
    }

    @Bean
    public KafkaTemplate<String, Object> kafkaTemplate(ProducerFactory<String, Object> producerFactory) {
        return new KafkaTemplate<>(producerFactory);
    }

    @Bean
    public ConsumerFactory<String, Object> consumerFactory(KafkaProperties properties, ObjectMapper objectMapper) {
        JsonDeserializer<Object> valueDeserializer = new JsonDeserializer<>(objectMapper);
        valueDeserializer.addTrustedPackages(TRUSTED_PACKAGE);
        Map<String, Object> consumerProps = properties.buildConsumerProperties();
        return new DefaultKafkaConsumerFactory<>(consumerProps, new StringDeserializer(), valueDeserializer);
    }

    @Bean
    public ConcurrentKafkaListenerContainerFactory<String, Object> kafkaListenerContainerFactory(
            ConsumerFactory<String, Object> consumerFactory) {
        ConcurrentKafkaListenerContainerFactory<String, Object> factory = new ConcurrentKafkaListenerContainerFactory<>();
        factory.setConsumerFactory(consumerFactory);
        return factory;
    }
}