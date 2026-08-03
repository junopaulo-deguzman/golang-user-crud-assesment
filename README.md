# Basic User CRUD API with Authentication

A small REST API written in Go for creating, retrieving, updating, and deleting users. It uses HTTP Basic Authentication, validates incoming JSON, stores users in MySQL, and returns JSON responses.

## Requirements

- Go 1.24 or newer
- Docker with Docker Compose
- `curl` for the examples below

## Setup

1. Create a `.env` file in the project root:

   ```dotenv
   BASIC_AUTH_USERNAME=admin
   BASIC_AUTH_PASSWORD=admin

   MYSQL_DATABASE=user_crud
   MYSQL_USER=user_crud
   MYSQL_PASSWORD=admin
   MYSQL_ROOT_PASSWORD=admin
   MYSQL_DSN=user_crud:admin@tcp(localhost:3306)/user_crud?parseTime=true
   ```

2. Start MySQL:

   ```bash
   docker compose up -d mysql
   ```

   The migration in `migrations/001_create_users.sql` runs automatically when the database volume is created for the first time.

3. Load the environment variables and start the API:

   ```bash
   set -a
   source .env
   set +a
   go run ./cmd/server
   ```

   The API listens on `http://localhost:8080`. Every exposed endpoint requires Basic Authentication.

## API endpoints

| Method | Path | Purpose | Success status |
| --- | --- | --- | --- |
| `POST` | `/users` | Create a user | `201 Created` |
| `GET` | `/users/{id}` | Retrieve a user | `200 OK` |
| `PUT` | `/users/{id}` | Replace a user's details | `200 OK` |
| `PATCH` | `/users/{id}` | Partially update a user | `200 OK` |
| `DELETE` | `/users/{id}` | Delete a user | `204 No Content` |

POST and PUT require complete user details. PATCH accepts at least one supported field and leaves omitted fields unchanged. Usernames and email addresses must be unique. See [`openapi.yaml`](openapi.yaml) for the complete request validation and response contract.

## Test with curl

Run these commands from a shell where the `.env` file has been loaded. Replace user ID `1` if the create response returns a different ID.

### Create a user

```bash
curl -i \
  -u "${BASIC_AUTH_USERNAME}:${BASIC_AUTH_PASSWORD}" \
  -H "Content-Type: application/json" \
  -d '{"username":"user","email":"user@example.com","age":30}' \
  http://localhost:8080/users
```

### Retrieve a user

```bash
curl -i \
  -u "${BASIC_AUTH_USERNAME}:${BASIC_AUTH_PASSWORD}" \
  http://localhost:8080/users/1
```

### Replace a user

```bash
curl -i \
  -X PUT \
  -u "${BASIC_AUTH_USERNAME}:${BASIC_AUTH_PASSWORD}" \
  -H "Content-Type: application/json" \
  -d '{"username":"user-updated","email":"user.updated@example.com","age":31}' \
  http://localhost:8080/users/1
```

### Patch a user

```bash
curl -i \
  -X PATCH \
  -u "${BASIC_AUTH_USERNAME}:${BASIC_AUTH_PASSWORD}" \
  -H "Content-Type: application/json" \
  -d '{"age":35}' \
  http://localhost:8080/users/1
```

### Delete a user

```bash
curl -i \
  -X DELETE \
  -u "${BASIC_AUTH_USERNAME}:${BASIC_AUTH_PASSWORD}" \
  http://localhost:8080/users/1
```

### Check authentication

This request omits credentials and should return `401 Unauthorized` with a JSON error:

```bash
curl -i http://localhost:8080/users/1
```

Errors use this response shape:

```json
{
  "error": "User not found"
}
```

## OpenAPI documentation

The complete API contract is in [`openapi.yaml`](openapi.yaml). To explore it interactively, open [Swagger Editor](https://editor.swagger.io/), select **File > Import File**, and choose `openapi.yaml`.

## Tests

```bash
go test ./...
```

## Stop the database

```bash
docker compose down
```

See [`ARCHITECTURE.md`](ARCHITECTURE.md) for the project structure and request flow.
