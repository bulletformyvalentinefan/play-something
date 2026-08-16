package com.spotify.group.domain.user;

import java.time.LocalDateTime;
import java.util.UUID;

public record UserResponse(
        UUID id,
        String nombre,
        String email,
        LocalDateTime fechaCreacion
) {
    public static UserResponse fromEntity(User user){
        return new UserResponse(
                user.getId(),
                user.getNombre(),
                user.getEmail(),
                user.getFechaCreacion()
        );
    }
}
