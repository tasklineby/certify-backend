# Swagger Documentation

## Generating Documentation

To generate/update the Swagger documentation, run:

```bash
# Install swag if not already installed
go install github.com/swaggo/swag/cmd/swag@latest

# Generate documentation
swag init -g cmd/main.go -o docs --parseDependency --parseInternal
```

## Viewing Documentation

Once the server is running, you can access the Swagger UI at:

```
http://localhost:8081/swagger/index.html
```

## Endpoints Documented

### Auth Endpoints (Public)
- `POST /api/auth/login` - Login user
- `POST /api/auth/register` - Register employee
- `POST /api/auth/refresh` - Refresh access token
- `POST /api/auth/logout` - Logout user

### User Endpoints

**Public:**
- `POST /api/user/company` - Create company with admin

**Protected (Require Bearer Token):**
- `GET /api/user/me` - Get current user
- `PUT /api/user/me` - Update current user
- `GET /api/user/{id}` - Get user by ID
- `PUT /api/user/{id}` - Update user by ID (admin only, same company)
- `DELETE /api/user/{id}` - Delete user by ID (admin only, same company)
- `GET /api/user/company` - Get users by company

