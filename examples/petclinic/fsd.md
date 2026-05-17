# Spring PetClinic Functional Specification

## FR-001 Owner Search

Staff can search for owners by last name from the owner search page.

Acceptance criteria:
- Search accepts a full or partial last name.
- Matching owners are listed with owner name, address, city, telephone, and pets.
- If one owner matches exactly, the owner detail page can be shown.

## FR-002 Owner Registration And Update

Staff can create a new owner and update an existing owner's contact details.

Acceptance criteria:
- Owner first name, last name, address, city, and telephone are captured.
- New owners are persisted and visible from owner search.
- Existing owner details can be edited and saved.

## FR-003 Pet Management

Staff can add and update pets for an owner.

Acceptance criteria:
- A pet has a name, birth date, and pet type.
- New pets are associated with the selected owner.
- Existing pet details can be edited without changing the owning owner.

## FR-004 Visit Recording

Staff can record a veterinary visit for a pet.

Acceptance criteria:
- A visit captures visit date and description.
- Visits are associated with the selected pet.
- Recorded visits are visible from the owner or pet detail workflow.

## FR-005 Veterinarian Directory

Users can view the veterinarian directory.

Acceptance criteria:
- Veterinarians are listed with their names.
- Veterinarian specialties are shown when present.
- The directory can be returned for browser and API consumers.
