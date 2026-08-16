package com.spotify.group.domain.user;

import org.springframework.transaction.annotation.Transactional;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

import java.util.UUID;

@Service
@RequiredArgsConstructor
public class UserService {

    private final UserRepository userRepository;

    @Transactional
    public UserResponse register(CreateUserRequest request) {
        if (userRepository.existsByEmail(request.email())) {
            throw new IllegalArgumentException("El email ya ha sido usado");
        }
        User user = new User();
        user.setNombre(request.nombre());
        user.setEmail(request.email());

        User saveUser = userRepository.save(user);
        return UserResponse.fromEntity(saveUser);
    }

    @Transactional(readOnly = true)
    public UserResponse login(LoginRequest request) {
        User user = userRepository.findByEmail(request.email()).orElseThrow(()-> new IllegalArgumentException("Usuario no encontrado"));
        return UserResponse.fromEntity(user);
    }

    @Transactional(readOnly = true)
    public UserResponse findById(UUID id) {
        User user = userRepository.findById(id).orElseThrow(() -> new IllegalArgumentException("Usuario no encotrado"));
        return UserResponse.fromEntity(user);
    }
}
