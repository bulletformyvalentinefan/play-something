package com.spotify.group.domain.user;

import org.springframework.data.jpa.repository.JpaRepository;

import java.util.Optional;
import java.util.UUID;

public interface UserRepository extends JpaRepository<User, UUID> {

    Optional<User> findByEmail(String email);  // Retorna un Optional con el User para login o búsquedas seguras
    boolean existsByEmail(String email); // Retorna true/false para verificar duplicados en el registro

}
