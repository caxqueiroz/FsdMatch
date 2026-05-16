package com.example.notebook.controller;

import jakarta.persistence.Entity;
import jakarta.persistence.Id;
import jakarta.persistence.Table;
import java.time.Instant;

@Entity
@Table(name = "notes")
public class Note {

    @Id
    private Long id;
    private String title;
    private String body;
    private Instant deletedAt;

    public Note() {}

    public Note(String title, String body) {
        this.title = title;
        this.body = body;
    }

    public Long id() { return id; }
    public String title() { return title; }
    public String body() { return body; }
}
