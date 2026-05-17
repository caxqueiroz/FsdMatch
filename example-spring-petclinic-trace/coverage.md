# Coverage — fsdtrace

_Run `match-1779029481` · generated 2026-05-17 14:58:38 UTC_

## Roll-up

| Section | Total | Implemented | Drifts | Missing |
|---|---:|---:|---:|---:|
| 3.1 Welcome / Home | 1 | 1 | 0 | 0 |
| 3.2 Owner Management | 7 | 6 | 1 | 0 |
| 3.3 Pet Management | 4 | 4 | 0 | 0 |
| 3.4 Visit Management | 3 | 3 | 0 | 0 |
| 3.5 Veterinarians | 3 | 2 | 1 | 0 |
| 3.6 Error Handling | 2 | 1 | 1 | 0 |
| 3.7 Internationalization (I18n) | 3 | 1 | 0 | 2 |
| **Total** | **23** | **18** | **3** | **2** |

## Section: 3.1 Welcome / Home

Total 1 · implemented 1 · drifts 0 · missing 0

### FR-HOME-1 — Display the welcome landing page at GET /. — IMPLEMENTED

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| implements | 0.95 |  | `rest_endpoint` GET / | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/WelcomeController.java:25-28` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:72-75` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:77-87` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/find | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:89-92` |
| unrelated | 0.00 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:136-139` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId} | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:99-104` |
| unrelated | 0.00 | ✓ (5) | `rest_endpoint` POST /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:106-127` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:84-87` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:91-102` |
| unrelated | 0.00 |  | `rest_endpoint` GET /oups | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/CrashController.java:31-35` |
| unrelated | 0.00 |  | `rest_endpoint` GET /vets.html | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:44-52` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /vets | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:69-76` |


**Evidence for `GET /`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/WelcomeController.java:25-28` — Defines GET / handler returning the 'welcome' view, which resolves to welcome.html in the Spring MVC view layer.

_Notes:_ Satisfies the acceptance criterion that GET / renders welcome.html.

## Section: 3.2 Owner Management

Total 7 · implemented 6 · drifts 1 · missing 0

### FR-OWN-1 — Provide owner search form by last-name prefix. — IMPLEMENTED

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| drifts | 0.72 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| implements | 0.95 | ✓ (1) | `rest_endpoint` GET /owners/find | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:89-92` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Owner (table=owners) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Owner.java:47-176` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:72-75` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:77-87` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:136-139` |
| unrelated | 0.00 | ✓ (4) | `rest_endpoint` POST /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:141-159` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId} | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:99-104` |
| unrelated | 0.00 | ✓ (5) | `rest_endpoint` POST /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:106-127` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` |
| unrelated | 0.00 | ✓ (3) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:134-160` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:84-87` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:91-102` |
| unrelated | 0.00 |  | `rest_endpoint` GET / | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/WelcomeController.java:25-28` |


**Evidence for `GET /owners`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-107` — Processes owner search input by reading Owner.lastName and searching by last name, but it is exposed at GET /owners rather than the required GET /owners/find search-form endpoint.

_Notes:_ Same owner last-name search surface, but the route and behavior are for processing search results rather than serving the required form.

**Evidence for `GET /owners/find`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:89-92` — Defines GET /owners/find and returns the owners/findOwners view, making the owner search form available at the required path.

### FR-OWN-2 — Submit owner search form and display results based on match count. — IMPLEMENTED

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| implements | 0.88 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:72-75` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:77-87` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/find | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:89-92` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:136-139` |
| unrelated | 0.00 | ✓ (4) | `rest_endpoint` POST /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:141-159` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId} | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:99-104` |
| unrelated | 0.00 | ✓ (5) | `rest_endpoint` POST /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:106-127` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` |
| unrelated | 0.00 | ✓ (3) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:134-160` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:84-87` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:91-102` |
| unrelated | 0.00 |  | `rest_endpoint` GET / | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/WelcomeController.java:25-28` |
| unrelated | 0.00 |  | `rest_endpoint` GET /vets.html | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:44-52` |


**Evidence for `GET /owners`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-104` — Defines GET /owners search handler, normalizes null lastName to empty string, and delegates to paginated last-name search.
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:105-118` — Implements match-count behavior: zero results adds notFound field error and redisplays find form; one result redirects to owner detail; multiple results renders via pagination model.

_Notes:_ The endpoint implements the core GET /owners match-count flow; repository method and page-size details are delegated to the helper not shown in this artifact.

### FR-OWN-3 — Render an empty owner form. — IMPLEMENTED

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| drifts | 0.82 | ✓ (2) | `rest_endpoint` POST /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:77-87` |
| implements | 0.95 | ✓ (1) | `rest_endpoint` GET /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:72-75` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Owner (table=owners) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Owner.java:47-176` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/find | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:89-92` |
| unrelated | 0.00 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:136-139` |
| unrelated | 0.00 | ✓ (4) | `rest_endpoint` POST /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:141-159` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId} | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:99-104` |
| unrelated | 0.00 | ✓ (5) | `rest_endpoint` POST /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:106-127` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` |
| unrelated | 0.00 | ✓ (3) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:134-160` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:84-87` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:91-102` |
| unrelated | 0.00 |  | `rest_endpoint` GET / | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/WelcomeController.java:25-28` |


**Evidence for `POST /owners/new`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:77-87` — Defines POST /owners/new to process owner creation and return the owner form only on validation errors, rather than GET /owners/new rendering an empty owner form.

_Notes:_ Same owner-creation path, but wrong HTTP method and behavior for the FR.

**Evidence for `GET /owners/new`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:72-75` — Defines GET /owners/new and returns the owner create/update form view.

_Notes:_ The endpoint matches the required path and renders the owner form view for creating a new owner.

### FR-OWN-4 — Validate and create a new owner via POST /owners/new. — IMPLEMENTED

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| implements | 0.99 | ✓ (2) | `rest_endpoint` POST /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:77-87` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Owner (table=owners) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Owner.java:47-176` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:72-75` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/find | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:89-92` |
| unrelated | 0.00 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:136-139` |
| unrelated | 0.00 | ✓ (4) | `rest_endpoint` POST /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:141-159` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId} | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:99-104` |
| unrelated | 0.00 | ✓ (5) | `rest_endpoint` POST /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:106-127` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` |
| unrelated | 0.00 | ✓ (3) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:134-160` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:84-87` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:91-102` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /vets | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:69-76` |


**Evidence for `POST /owners/new`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:77-87` — POST /owners/new validates Owner with @Valid, redisplays the owner form on BindingResult errors, saves the owner on success, and redirects to /owners/{newOwnerId} using owner.getId().

### FR-OWN-5 — Render an edit form pre-populated with existing owner data. — DRIFTS

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| drifts | 0.88 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:136-139` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Owner (table=owners) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Owner.java:47-176` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:72-75` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:77-87` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/find | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:89-92` |
| unrelated | 0.00 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| unrelated | 0.00 | ✓ (4) | `rest_endpoint` POST /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:141-159` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId} | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:99-104` |
| unrelated | 0.00 | ✓ (5) | `rest_endpoint` POST /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:106-127` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` |
| unrelated | 0.00 | ✓ (3) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:134-160` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:84-87` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:91-102` |
| unrelated | 0.00 |  | `rest_endpoint` GET / | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/WelcomeController.java:25-28` |


**Evidence for `GET /owners/{ownerId}/edit`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:136-139` — Defines the required GET /owners/{ownerId}/edit endpoint and returns the owner create/update form view, but the shown method does not load the existing owner or add owner data to the model for pre-population.

_Notes:_ Exact endpoint and form rendering are present, but pre-population with existing owner data is not evidenced in this artifact.

### FR-OWN-6 — Validate and persist owner edits then redirect to the owner detail page. — IMPLEMENTED

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| drifts | 0.86 | ✓ (4) | `rest_endpoint` POST /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:141-159` |
| implements | 0.58 |  | `entity` org.springframework.samples.petclinic.owner.Owner (table=owners) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Owner.java:47-176` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:72-75` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:77-87` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/find | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:89-92` |
| unrelated | 0.00 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:136-139` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId} | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:99-104` |
| unrelated | 0.00 | ✓ (5) | `rest_endpoint` POST /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:106-127` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` |
| unrelated | 0.00 | ✓ (3) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:134-160` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Visit (table=visits) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Visit.java:34-68` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:84-87` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:91-102` |


**Evidence for `POST /owners/{ownerId}/edit`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:141-159` — Defines POST /owners/{ownerId}/edit, validates via @Valid/BindingResult, saves the owner, and redirects to /owners/{ownerId}, but also relies on a bound owner.id mismatch check rather than rejecting id binding via @InitBinder.

_Notes:_ Same endpoint and core update flow, but anti-tampering behavior drifts from the FR: the form id is compared/accepted if matching instead of being rejected by @InitBinder.

**Evidence for `org.springframework.samples.petclinic.owner.Owner (table=owners)`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Owner.java:51-62` — Owner fields have Bean Validation constraints (@NotBlank and telephone @Pattern), supporting validation of submitted owner edits.

_Notes:_ Implements the validation aspect of owner edits, though the POST edit controller, persistence, redirect, path-authoritative behavior, and InitBinder anti-tampering are not shown in this artifact.

### FR-OWN-7 — Render owner details with pets and visits. — IMPLEMENTED

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| implements | 0.90 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId} | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Owner (table=owners) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Owner.java:47-176` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:72-75` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:77-87` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/find | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:89-92` |
| unrelated | 0.00 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:136-139` |
| unrelated | 0.00 | ✓ (4) | `rest_endpoint` POST /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:141-159` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:99-104` |
| unrelated | 0.00 | ✓ (5) | `rest_endpoint` POST /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:106-127` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` |
| unrelated | 0.00 | ✓ (3) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:134-160` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:84-87` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:91-102` |
| unrelated | 0.00 |  | `rest_endpoint` GET / | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/WelcomeController.java:25-28` |


**Evidence for `GET /owners/{ownerId}`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` — Defines GET /owners/{ownerId}, creates ModelAndView for owners/ownerDetails, loads the Owner by id, adds it to the model, and returns the view.

_Notes:_ Implements the required owner details endpoint and renders the owner details view with the owner model object.

## Section: 3.3 Pet Management

Total 4 · implemented 4 · drifts 0 · missing 0

### FR-PET-1 — Render a new pet form for a given owner. — IMPLEMENTED

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| drifts | 0.76 | ✓ (5) | `rest_endpoint` POST /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:106-127` |
| implements | 0.86 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:99-104` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Owner (table=owners) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Owner.java:47-176` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:72-75` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:77-87` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/find | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:89-92` |
| unrelated | 0.00 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:136-139` |
| unrelated | 0.00 | ✓ (4) | `rest_endpoint` POST /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:141-159` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId} | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` |
| unrelated | 0.00 | ✓ (3) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:134-160` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:84-87` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:91-102` |
| unrelated | 0.00 |  | `rest_endpoint` GET / | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/WelcomeController.java:25-28` |


**Evidence for `POST /owners/{ownerId}/pets/new`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:106-127` — Defines POST handling for /pets/new in the owner pet creation flow, returning the pet create/update form only on validation errors and otherwise saving/redirecting; it does not implement the required GET form-rendering endpoint or show pet type population.

_Notes:_ Same owner pet creation surface, but different HTTP method/behavior from required GET /owners/{ownerId}/pets/new.

**Evidence for `GET /owners/{ownerId}/pets/new`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:99-104` — Defines GET /owners/{ownerId}/pets/new, creates a new Pet in the specified Owner context, and returns the pet create/update form view.

_Notes:_ Implements the form-rendering criterion for creating a new pet for an owner; dropdown population/order is not evidenced in this artifact.

### FR-PET-2 — Create and attach a new pet to an owner. — IMPLEMENTED

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| implements | 0.99 | ✓ (5) | `rest_endpoint` POST /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:106-127` |
| implements | 0.78 |  | `entity` org.springframework.samples.petclinic.owner.Owner (table=owners) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Owner.java:47-176` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:72-75` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:77-87` |
| unrelated | 0.00 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:136-139` |
| unrelated | 0.00 | ✓ (4) | `rest_endpoint` POST /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:141-159` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId} | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Pet (table=pets) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Pet.java:44-85` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:99-104` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` |
| unrelated | 0.00 | ✓ (3) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:134-160` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.PetType (table=types) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetType.java:26-30` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:84-87` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:91-102` |


**Evidence for `POST /owners/{ownerId}/pets/new`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:106-127` — POST creation handler validates @Valid Pet and additional constraints, checks BindingResult, attaches pet to owner via owner.addPet(pet), persists owner via owners.save(owner), and redirects to /owners/{ownerId}.

**Evidence for `org.springframework.samples.petclinic.owner.Owner (table=owners)`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Owner.java:64-67` — Owner.pets is mapped with CascadeType.ALL, enabling persistence of an owner to cascade to attached pets.
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Owner.java:97-99` — Owner.addPet(Pet) attaches a new pet to the owner's pets collection.

_Notes:_ Satisfies the attachment and cascading persistence aspects of creating a pet for an owner, though the POST endpoint behavior itself is not shown.

### FR-PET-3 — Render the pet edit form pre-populated with existing data. — IMPLEMENTED

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| implements | 0.86 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:72-75` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:77-87` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/find | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:89-92` |
| unrelated | 0.00 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:136-139` |
| unrelated | 0.00 | ✓ (4) | `rest_endpoint` POST /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:141-159` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId} | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:99-104` |
| unrelated | 0.00 | ✓ (5) | `rest_endpoint` POST /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:106-127` |
| unrelated | 0.00 | ✓ (3) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:134-160` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:84-87` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:91-102` |
| unrelated | 0.00 |  | `rest_endpoint` GET / | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/WelcomeController.java:25-28` |
| unrelated | 0.00 |  | `rest_endpoint` GET /vets.html | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:44-52` |


**Evidence for `GET /owners/{ownerId}/pets/{petId}/edit`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` — Defines the GET /pets/{petId}/edit endpoint under the owner route and returns the pet create/update form view.

_Notes:_ Implements the rendering aspect of GET /owners/{ownerId}/pets/{petId}/edit; the shown method returns the pet form view, though pre-population is not explicit in this snippet.

### FR-PET-4 — Validate and persist pet updates, then redirect to the owner page. — IMPLEMENTED

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| drifts | 0.86 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` |
| drifts | 0.72 | ✓ (5) | `rest_endpoint` POST /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:106-127` |
| implements | 0.92 | ✓ (3) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:134-160` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Owner (table=owners) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Owner.java:47-176` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:77-87` |
| unrelated | 0.00 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:136-139` |
| unrelated | 0.00 | ✓ (4) | `rest_endpoint` POST /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:141-159` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId} | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Pet (table=pets) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Pet.java:44-85` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:99-104` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.PetType (table=types) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetType.java:26-30` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Visit (table=visits) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Visit.java:34-68` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:84-87` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:91-102` |


**Evidence for `GET /owners/{ownerId}/pets/{petId}/edit`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` — Defines the pet edit route for GET and returns the create/update form, but does not implement the required POST validation, persistence, or redirect behavior.

_Notes:_ Same pet edit surface area, but this is only the form initialization endpoint rather than POST /owners/{ownerId}/pets/{petId}/edit.

**Evidence for `POST /owners/{ownerId}/pets/new`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:106-107` — Defines the pet creation POST endpoint rather than the required pet edit endpoint.
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:110-112` — Performs similar duplicate-name validation for new pets, rejecting the name field with code duplicate.
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:115-117` — Performs similar future birthDate rejection using typeMismatch.birthDate.
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:124-127` — Persists via owner.addPet/save and redirects to /owners/{ownerId}, but for creation rather than update.

_Notes:_ Similar pet validation and redirect behavior exists on the new-pet endpoint, but it is not the required edit endpoint.

**Evidence for `POST /owners/{ownerId}/pets/{petId}/edit`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:134-136` — Defines the POST pet edit endpoint and applies @Valid to the submitted Pet.
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:141-145` — Checks for duplicate pet names within the owner and rejects the name field with code duplicate.
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:148-150` — Rejects a future birthDate with field error typeMismatch.birthDate.
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:153-159` — Returns the form on validation errors; otherwise updates pet details and redirects to /owners/{ownerId}.

_Notes:_ Implements the targeted pet update POST flow with validation, duplicate-name handling, future-date rejection, update, and owner-page redirect.

## Section: 3.4 Visit Management

Total 3 · implemented 3 · drifts 0 · missing 0

### FR-VIS-1 — Render a new visit form pre-filled with today's date for a given pet. — IMPLEMENTED

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| implements | 0.83 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:84-87` |
| implements | 0.78 |  | `entity` org.springframework.samples.petclinic.owner.Visit (table=visits) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Visit.java:34-68` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:72-75` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:77-87` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/find | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:89-92` |
| unrelated | 0.00 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:136-139` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId} | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Pet (table=pets) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Pet.java:44-85` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:99-104` |
| unrelated | 0.00 | ✓ (5) | `rest_endpoint` POST /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:106-127` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` |
| unrelated | 0.00 | ✓ (3) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:134-160` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:91-102` |
| unrelated | 0.00 |  | `rest_endpoint` GET / | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/WelcomeController.java:25-28` |


**Evidence for `GET /owners/{ownerId}/pets/{petId}/visits/new`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:84-87` — Defines GET /owners/{ownerId}/pets/{petId}/visits/new and returns the visit creation/update form view.

_Notes:_ Matches the required endpoint and renders the visit form; the cited artifact does not itself show the today's-date initialization.

**Evidence for `org.springframework.samples.petclinic.owner.Visit (table=visits)`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Visit.java:45-50` — The Visit constructor initializes the visit date to LocalDate.now(), providing the default value needed for a new visit form to be pre-filled with today's date.

_Notes:_ Implements the date prefill aspect of the new visit form requirement, though the GET endpoint itself is not shown among the candidates.

### FR-VIS-2 — Create a new visit for a pet and redirect to the owner page. — IMPLEMENTED

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| implements | 0.99 | ✓ (2) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:91-102` |
| implements | 0.86 |  | `entity` org.springframework.samples.petclinic.owner.Pet (table=pets) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Pet.java:44-85` |
| implements | 0.72 |  | `entity` org.springframework.samples.petclinic.owner.Owner (table=owners) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Owner.java:47-176` |
| implements | 0.67 |  | `entity` org.springframework.samples.petclinic.owner.Visit (table=visits) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Visit.java:34-68` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:72-75` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:77-87` |
| unrelated | 0.00 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:136-139` |
| unrelated | 0.00 | ✓ (4) | `rest_endpoint` POST /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:141-159` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId} | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:99-104` |
| unrelated | 0.00 | ✓ (5) | `rest_endpoint` POST /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:106-127` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` |
| unrelated | 0.00 | ✓ (3) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:134-160` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:84-87` |


**Evidence for `POST /owners/{ownerId}/pets/{petId}/visits/new`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:91-102` — POST handler for /owners/{ownerId}/pets/{petId}/visits/new validates @Valid Visit with BindingResult, attaches it via owner.addVisit(petId, visit), saves the owner, and redirects to /owners/{ownerId}.

**Evidence for `org.springframework.samples.petclinic.owner.Pet (table=pets)`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Pet.java:56-59` — Defines the pet-to-visit relationship with CascadeType.ALL, allowing visits attached to a pet to be persisted through the pet.
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Pet.java:81-82` — Provides addVisit(Visit) which attaches a visit to the pet's visit collection.

_Notes:_ Satisfies the visit attachment criterion and the pet-to-visit cascade portion of persistence.

**Evidence for `org.springframework.samples.petclinic.owner.Owner (table=owners)`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Owner.java:64-67` — Defines the owner-to-pet relationship with CascadeType.ALL, supporting persistence of owned pets when the owner is saved.

_Notes:_ Implements the owner-side cascade needed for saving an owner to propagate to contained pets; combined with the pet-to-visit cascade this supports cascading to visits.

**Evidence for `org.springframework.samples.petclinic.owner.Visit (table=visits)`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Visit.java:42-43` — Declares a validation constraint requiring the visit description to be non-blank.

_Notes:_ Defines bean-validation metadata for Visit, supporting the visit validation acceptance criterion.

### FR-VIS-3 — Owner detail page lists visits per pet chronologically. — IMPLEMENTED

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| drifts | 0.62 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId} | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` |
| implements | 0.86 |  | `entity` org.springframework.samples.petclinic.owner.Pet (table=pets) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Pet.java:44-85` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Owner (table=owners) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Owner.java:47-176` |
| unrelated | 0.00 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:99-104` |
| unrelated | 0.00 | ✓ (5) | `rest_endpoint` POST /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:106-127` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` |
| unrelated | 0.00 | ✓ (3) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:134-160` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.PetType (table=types) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetType.java:26-30` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Visit (table=visits) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Visit.java:34-68` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:84-87` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:91-102` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.vet.Vet (table=vets) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/Vet.java:43-74` |
| unrelated | 0.00 |  | `rest_endpoint` GET /vets.html | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:44-52` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /vets | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:69-76` |


**Evidence for `GET /owners/{ownerId}`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` — Owner detail endpoint renders owners/ownerDetails and adds the Owner to the model, but this artifact does not show visits being listed per pet or ordered chronologically.

_Notes:_ Same owner detail page surface area, but the artifact alone does not demonstrate chronological visit listing.

**Evidence for `org.springframework.samples.petclinic.owner.Pet (table=pets)`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Pet.java:56-59` — Pet owns a collection of visits and orders them by visit date ascending, providing chronological visits per pet.
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Pet.java:77-79` — The ordered visits collection is exposed through getVisits(), enabling the owner detail page to list recorded visits for each pet.

## Section: 3.5 Veterinarians

Total 3 · implemented 2 · drifts 1 · missing 0

### FR-VET-1 — Render a paginated veterinarians table. — IMPLEMENTED

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| implements | 0.78 |  | `rest_endpoint` GET /vets.html | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:44-52` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Owner (table=owners) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Owner.java:47-176` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/find | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:89-92` |
| unrelated | 0.00 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:99-104` |
| unrelated | 0.00 | ✓ (5) | `rest_endpoint` POST /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:106-127` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` |
| unrelated | 0.00 | ✓ (3) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:134-160` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.PetType (table=types) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetType.java:26-30` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:84-87` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:91-102` |
| unrelated | 0.00 |  | `rest_endpoint` GET / | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/WelcomeController.java:25-28` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.vet.Specialty (table=specialties) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/Specialty.java:28-32` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.vet.Vet (table=vets) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/Vet.java:43-74` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /vets | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:69-76` |


**Evidence for `GET /vets.html`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:44-52` — GET /vets.html handler accepts a page parameter, obtains a Page<Vet> via findPaginated(page), adds the page contents to the vets model object, and returns the pagination model for rendering.

_Notes:_ Implements the anchored /vets.html endpoint and paginated retrieval path, though the provided snippet does not show the page size or table template.

### FR-VET-2 — Return complete list of veterinarians via GET /vets. — IMPLEMENTED

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| implements | 0.98 | ✓ (1) | `rest_endpoint` GET /vets | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:69-76` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Owner (table=owners) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Owner.java:47-176` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/find | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:89-92` |
| unrelated | 0.00 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId} | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Pet (table=pets) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Pet.java:44-85` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:99-104` |
| unrelated | 0.00 | ✓ (5) | `rest_endpoint` POST /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:106-127` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` |
| unrelated | 0.00 | ✓ (3) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:134-160` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:84-87` |
| unrelated | 0.00 |  | `rest_endpoint` GET / | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/WelcomeController.java:25-28` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.vet.Specialty (table=specialties) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/Specialty.java:28-32` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.vet.Vet (table=vets) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/Vet.java:43-74` |
| unrelated | 0.00 |  | `rest_endpoint` GET /vets.html | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:44-52` |


**Evidence for `GET /vets`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:69-76` — Defines GET /vets endpoint returning @ResponseBody Vets wrapper populated with all vets from vetRepository.findAll().

_Notes:_ Satisfies programmatic GET /vets returning the complete vet list in a Vets wrapper; Spring response body enables JSON/XML negotiation via configured converters.

### FR-VET-3 — Vet lookups shall be cached using a Caffeine cache. — DRIFTS

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| drifts | 0.78 | ✓ (1) | `rest_endpoint` GET /vets | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:69-76` |
| drifts | 0.66 |  | `rest_endpoint` GET /vets.html | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:44-52` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Owner (table=owners) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Owner.java:47-176` |
| unrelated | 0.00 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Pet (table=pets) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Pet.java:44-85` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:99-104` |
| unrelated | 0.00 | ✓ (5) | `rest_endpoint` POST /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:106-127` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` |
| unrelated | 0.00 | ✓ (3) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:134-160` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.PetType (table=types) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetType.java:26-30` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Visit (table=visits) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Visit.java:34-68` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:84-87` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:91-102` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.vet.Specialty (table=specialties) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/Specialty.java:28-32` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.vet.Vet (table=vets) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/Vet.java:43-74` |


**Evidence for `GET /vets`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:69-76` — GET /vets performs a vet lookup by calling vetRepository.findAll() and returning the results, but no Caffeine cache named 'vets' is used.

_Notes:_ Same vet lookup surface, but it bypasses the required Caffeine cache.

**Evidence for `GET /vets.html`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:44-52` — GET /vets.html retrieves vets via findPaginated(page) and populates the model, but no Caffeine cache named 'vets' is shown.

_Notes:_ Vet list lookup behavior exists, but caching is not implemented in this artifact.

## Section: 3.6 Error Handling

Total 2 · implemented 1 · drifts 1 · missing 0

### FR-ERR-1 — Render unhandled exceptions using the standard error page. — DRIFTS

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| drifts | 0.72 |  | `rest_endpoint` GET /oups | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/CrashController.java:31-35` |
| drifts | 0.66 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId} | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:72-75` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:77-87` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/find | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:89-92` |
| unrelated | 0.00 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:136-139` |
| unrelated | 0.00 | ✓ (4) | `rest_endpoint` POST /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:141-159` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` |
| unrelated | 0.00 | ✓ (3) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:134-160` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:84-87` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:91-102` |
| unrelated | 0.00 |  | `rest_endpoint` GET / | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/WelcomeController.java:25-28` |
| unrelated | 0.00 |  | `rest_endpoint` GET /vets.html | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:44-52` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /vets | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:69-76` |


**Evidence for `GET /oups`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/CrashController.java:31-35` — Endpoint deliberately throws an unhandled RuntimeException, exercising the error-handling path, but it does not itself render error.html or add somethingHappened/error.* localized messages.

_Notes:_ Covers the unhandled-exception surface area but lacks the required standard error-page rendering behavior.

**Evidence for `GET /owners/{ownerId}`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` — Missing owner path throws an IllegalArgumentException with a specific raw message rather than explicitly rendering error.html with somethingHappened and localized error text.

_Notes:_ This overlaps with unhandled exception behavior, but the artifact does not implement the required standard error page rendering or localized error keys.

### FR-ERR-2 — CrashController provides a deliberate exception endpoint for testing error handling. — IMPLEMENTED

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| implements | 0.99 |  | `rest_endpoint` GET /oups | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/CrashController.java:31-35` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:72-75` |
| unrelated | 0.00 | ✓ (2) | `rest_endpoint` POST /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:77-87` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/find | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:89-92` |
| unrelated | 0.00 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:136-139` |
| unrelated | 0.00 | ✓ (4) | `rest_endpoint` POST /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:141-159` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId} | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:99-104` |
| unrelated | 0.00 | ✓ (5) | `rest_endpoint` POST /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:106-127` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:84-87` |
| unrelated | 0.00 |  | `rest_endpoint` GET / | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/WelcomeController.java:25-28` |
| unrelated | 0.00 |  | `rest_endpoint` GET /vets.html | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:44-52` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /vets | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:69-76` |


**Evidence for `GET /oups`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/CrashController.java:31-35` — GET /oups endpoint deliberately throws a RuntimeException to exercise error-handling behavior.

## Section: 3.7 Internationalization (I18n)

Total 3 · implemented 1 · drifts 0 · missing 2

### FR-I18N-1 — Source all user-facing strings from the messages resource bundle. — IMPLEMENTED

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| drifts | 0.96 | ✓ (2) | `rest_endpoint` POST /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:77-87` |
| drifts | 0.88 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| drifts | 0.72 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId} | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` |
| drifts | 0.62 |  | `rest_endpoint` GET /oups | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/CrashController.java:31-35` |
| implements | 0.82 |  | `entity` org.springframework.samples.petclinic.owner.Owner (table=owners) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Owner.java:47-176` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:72-75` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/find | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:89-92` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:136-139` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:99-104` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/visits/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/VisitController.java:84-87` |
| unrelated | 0.00 |  | `rest_endpoint` GET / | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/WelcomeController.java:25-28` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.vet.Specialty (table=specialties) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/Specialty.java:28-32` |
| unrelated | 0.00 |  | `rest_endpoint` GET /vets.html | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:44-52` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /vets | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:69-76` |


**Evidence for `POST /owners/new`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:80-85` — Hardcoded flash messages are user-facing strings and are not sourced from the messages/messages resource bundle.

_Notes:_ Uses literal user-facing success/error messages instead of message bundle keys/resolution.

**Evidence for `GET /owners`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:106-108` — The validation rejection includes a hardcoded default user-facing message, "not found", instead of sourcing it from the messages bundle.

_Notes:_ Although an error code is supplied, the hardcoded default message is not sourced from the resource bundle.

**Evidence for `GET /owners/{ownerId}`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:170-171` — Hardcoded owner-not-found error text may be surfaced to users and is not sourced from the messages/messages resource bundle.

_Notes:_ Exception message is a user-visible string candidate and should be internationalized if displayed.

**Evidence for `GET /oups`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/CrashController.java:33-34` — The exception message is hardcoded in the controller rather than sourced from the messages bundle.

_Notes:_ If surfaced on the error page, this is user-facing text and should be externalized.

**Evidence for `org.springframework.samples.petclinic.owner.Owner (table=owners)`:**
- `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Owner.java:59-61` — Validation message is externalized via the message key {telephone.invalid} rather than hardcoded text.

_Notes:_ The Bean Validation message uses a bundle key, consistent with sourcing user-facing validation text from messages.

### FR-I18N-2 — The application shall ship translations for specified locales. — MISSING

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Owner (table=owners) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Owner.java:47-176` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:72-75` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/find | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:89-92` |
| unrelated | 0.00 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:136-139` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId} | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Pet (table=pets) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Pet.java:44-85` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.PetType (table=types) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetType.java:26-30` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Visit (table=visits) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Visit.java:34-68` |
| unrelated | 0.00 |  | `rest_endpoint` GET /oups | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/CrashController.java:31-35` |
| unrelated | 0.00 |  | `rest_endpoint` GET / | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/WelcomeController.java:25-28` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.vet.Specialty (table=specialties) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/Specialty.java:28-32` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.vet.Vet (table=vets) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/Vet.java:43-74` |
| unrelated | 0.00 |  | `rest_endpoint` GET /vets.html | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:44-52` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /vets | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:69-76` |


### FR-I18N-3 — Verify locale files have the same keys as the default bundle. — MISSING

| Verdict | Confidence | Tested | Artifact | Location |
|---|---:|:---:|---|---|
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Owner (table=owners) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Owner.java:47-176` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:72-75` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/find | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:89-92` |
| unrelated | 0.00 |  | `rest_endpoint` GET /owners | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:94-119` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:136-139` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId} | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/OwnerController.java:166-174` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Pet (table=pets) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Pet.java:44-85` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/new | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:99-104` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /owners/{ownerId}/pets/{petId}/edit | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetController.java:129-132` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.PetType (table=types) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/PetType.java:26-30` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.owner.Visit (table=visits) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/owner/Visit.java:34-68` |
| unrelated | 0.00 |  | `rest_endpoint` GET /oups | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/system/CrashController.java:31-35` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.vet.Specialty (table=specialties) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/Specialty.java:28-32` |
| unrelated | 0.00 |  | `entity` org.springframework.samples.petclinic.vet.Vet (table=vets) | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/Vet.java:43-74` |
| unrelated | 0.00 | ✓ (1) | `rest_endpoint` GET /vets | `/Users/cq/Dev/spring-petclinic/src/main/java/org/springframework/samples/petclinic/vet/VetController.java:69-76` |


