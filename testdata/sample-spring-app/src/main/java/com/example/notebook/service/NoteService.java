package com.example.notebook.service;

import com.example.notebook.controller.Note;
import com.example.notebook.controller.NoteCreate;
import com.example.notebook.repository.NoteRepository;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
public class NoteService {

    private final NoteRepository repo;
    private final KafkaTemplate<String, Object> kafka;

    public NoteService(NoteRepository repo, KafkaTemplate<String, Object> kafka) {
        this.repo = repo;
        this.kafka = kafka;
    }

    public Note findById(Long id) {
        return repo.findById(id).orElse(null);
    }

    @Transactional
    public Note create(NoteCreate body) {
        Note saved = repo.save(new Note(body.title(), body.body()));
        kafka.send("notes-events", "notes.created", saved.id());
        return saved;
    }

    @Transactional
    public void softDelete(Long id) {
        repo.softDelete(id);
    }
}
