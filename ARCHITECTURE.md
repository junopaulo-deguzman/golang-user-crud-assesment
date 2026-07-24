## User Management API

### API endpoints

All endpoints require authentication

- `POST /users`: Create a new user
- `GET /users/{id}`: Retrieve user details by ID
- `PUT /users/{id}`: Udpate user details by ID
- `DELETE /users/{id}`: Delete user by ID

#### User Data Structure

- `id`: UUID
- `username`
- `email`
- `age` 


### Database

- Use MySQL
- Create `users` table
- Set up a connection pool to manage database connections

### Error Handling

- Handle errors gracefully and return appropriate HTTP status codes.
- Return JSON responses with error messages.
