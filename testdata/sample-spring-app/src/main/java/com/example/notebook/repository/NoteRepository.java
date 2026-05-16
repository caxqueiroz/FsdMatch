package com.example.notebook.repository;

import com.example.notebook.controller.Note;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.stereotype.Repository;

@Repository
public interface NoteRepository extends JpaRepository<Note, Long> {

    @Modifying
    @Query("UPDATE Note n SET n.deletedAt = CURRENT_TIMESTAMP WHERE n.id = :id")
    void softDelete(Long id);
}
