# Backend - Go REST API

A high-performance REST API built with Go and Gin framework, featuring Firebase integration, JWT authentication, and AI service connectivity.

## 🚀 Features

- **High Performance**: Built with Go and Gin framework
- **Authentication**: Firebase Authentication integration
- **Database**: Firebase Firestore for data persistence
- **File Storage**: Firebase Storage for file uploads
- **AI Integration**: OpenRouter and Replicate API integration
- **CORS Support**: Configurable cross-origin resource sharing
- **Middleware**: Request logging, authentication, and error handling
- **Docker Support**: Containerized deployment ready
- **Environment Configuration**: Flexible environment-based configuration

## 🛠️ Tech Stack

- **Language**: [Go 1.23+](https://golang.org/)
- **Framework**: [Gin Web Framework](https://gin-gonic.com/)
- **Database**: [Firebase Firestore](https://firebase.google.com/docs/firestore)
- **Authentication**: [Firebase Auth](https://firebase.google.com/docs/auth)
- **Storage**: [Firebase Storage](https://firebase.google.com/docs/storage)
- **AI Services**: OpenRouter API, Replicate API
- **Containerization**: [Docker](https://www.docker.com/)

## 📋 Prerequisites

- Go 1.23 or later
- Firebase project with Firestore, Authentication, and Storage enabled
- OpenRouter API key (optional, for AI features)
- Replicate API key (optional, for AI features)

## ⚡ Quick Start

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd backend
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Set up environment variables**
   ```bash
   cp .env.example .env
   ```
   
   Fill in your configuration in `.env`:
   ```env
   PORT=8080
   FIREBASE_PROJECT_ID=your_project_id
   FIREBASE_STORAGE_BUCKET=your_project.firebasestorage.app
   GOOGLE_APPLICATION_CREDENTIALS=serviceAccountKey.json
   OPENROUTER_API_KEY=your_openrouter_api_key
   REPLICATE_API_KEY=your_replicate_api_key
   ```

4. **Set up Firebase Service Account**
   - Go to Firebase Console → Project Settings → Service Accounts
   - Generate a new private key
   - Save the JSON file as `serviceAccountKey.json` in the backend directory
   - **Never commit this file to version control**

5. **Run the development server**
   ```bash
   go run main.go
   ```

6. **Test the API**
   Navigate to [http://localhost:8080](http://localhost:8080)

## 📁 Project Structure

```
backend/
├── cmd/                  # Application entry points
├── controllers/          # Request handlers
│   ├── auth.go           # Authentication controllers
│   ├── chat.go           # Chat-related controllers
│   └── user.go           # User management controllers
├── firebase/             # Firebase configuration and clients
│   ├── firebase.go       # Firebase initialization
│   └── storage.go        # Firebase Storage utilities
├── middleware/           # HTTP middleware
│   ├── auth.go           # Authentication middleware
│   └── cors.go           # CORS middleware
├── models/               # Data models and schemas
│   ├── user.go           # User model
│   ├── chat.go           # Chat model
│   └── message.go        # Message model
├── routes/               # Route definitions
│   └── routes.go         # Main route setup
├── services/             # Business logic services
│   ├── ai.go             # AI service integrations
│   ├── chat.go           # Chat service
│   └── user.go           # User service
├── temp/                 # Temporary file storage
├── main.go               # Application entry point
├── go.mod                # Go module definition
├── go.sum                # Go module checksums
├── Dockerfile            # Docker configuration
└── .dockerignore         # Docker ignore file
```

## 🔧 Available Commands

- `go run main.go` - Start development server
- `go build` - Build the application
- `go test ./...` - Run all tests
- `go mod tidy` - Clean up dependencies
- `docker build -t backend .` - Build Docker image
- `docker run -p 8080:8080 backend` - Run Docker container

## 🔐 Environment Variables

Create a `.env` file in the root directory with the following variables:

```env
# Server Configuration
PORT=8080

# Firebase Configuration
FIREBASE_PROJECT_ID=your_project_id
FIREBASE_STORAGE_BUCKET=your_project.firebasestorage.app
GOOGLE_APPLICATION_CREDENTIALS=serviceAccountKey.json

# AI Service APIs (Optional)
OPENROUTER_API_KEY=your_openrouter_api_key
REPLICATE_API_KEY=your_replicate_api_key

# Environment
GO_ENV=development
```

## 🔑 Firebase Setup

1. **Create a Firebase Project**
   - Go to [Firebase Console](https://console.firebase.google.com/)
   - Create a new project or use existing one

2. **Enable Required Services**
   - Authentication (Email/Password)
   - Firestore Database
   - Storage

3. **Generate Service Account Key**
   - Project Settings → Service Accounts
   - Generate new private key
   - Download JSON file as `serviceAccountKey.json`

4. **Security Rules**
   Update Firestore and Storage security rules as needed for your use case.

## 🔗 API Endpoints

### Authentication
- `POST /api/auth/register` - Register new user
- `POST /api/auth/login` - User login
- `POST /api/auth/logout` - User logout
- `GET /api/auth/verify` - Verify JWT token

### Users
- `GET /api/users/profile` - Get user profile
- `PUT /api/users/profile` - Update user profile
- `POST /api/users/avatar` - Upload user avatar

### Chat
- `GET /api/chat/conversations` - Get user conversations
- `POST /api/chat/conversations` - Create new conversation
- `GET /api/chat/conversations/:id/messages` - Get conversation messages
- `POST /api/chat/conversations/:id/messages` - Send new message

### AI Services
- `POST /api/ai/chat` - Chat with AI
- `POST /api/ai/generate` - Generate content with AI

## 🔒 Authentication

The API uses Firebase Authentication with JWT tokens:

1. **Client Authentication**
   - Frontend sends Firebase ID token in Authorization header
   - Backend verifies token with Firebase Admin SDK

2. **Protected Routes**
   - Use `AuthMiddleware` to protect routes
   - Middleware validates Firebase tokens
   - User information available in request context

3. **Example Usage**
   ```bash
   curl -H "Authorization: Bearer YOUR_FIREBASE_ID_TOKEN" \
        http://localhost:8080/api/users/profile
   ```

## 🐳 Docker Deployment

1. **Build the image**
   ```bash
   docker build -t your-app-backend .
   ```

2. **Run the container**
   ```bash
   docker run -p 8080:8080 \
     -e FIREBASE_PROJECT_ID=your_project_id \
     -e FIREBASE_STORAGE_BUCKET=your_bucket \
     -v /path/to/serviceAccountKey.json:/app/serviceAccountKey.json \
     your-app-backend
   ```

## 🚀 Production Deployment

### Environment Setup
1. Set environment variables on your hosting platform
2. Upload `serviceAccountKey.json` securely (not in repository)
3. Configure firewall rules for port 8080
4. Set up reverse proxy (nginx) if needed

### Recommended Platforms
- **Google Cloud Run** (Recommended for Firebase integration)
- **AWS ECS/Lambda**
- **DigitalOcean App Platform**
- **Railway**
- **Heroku**

## 🧪 Testing

Run tests with:
```bash
go test ./...
```

Run tests with coverage:
```bash
go test -cover ./...
```

Run specific package tests:
```bash
go test ./controllers
```

## 📊 Monitoring & Logging

The API includes:
- Request/response logging via Gin middleware
- Error handling and logging
- Performance metrics (add monitoring tools as needed)

For production, consider adding:
- Prometheus metrics
- Distributed tracing
- Health check endpoints

## 🔧 Configuration

### CORS Configuration
Update CORS settings in `main.go`:
```go
router.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"https://yourdomain.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
    AllowCredentials: true,
}))
```

### Rate Limiting
Consider adding rate limiting middleware for production use.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Commit your changes: `git commit -m 'Add amazing feature'`
4. Push to the branch: `git push origin feature/amazing-feature`
5. Open a Pull Request

### Code Style
- Follow Go best practices and conventions
- Use `gofmt` for code formatting
- Add comments for exported functions
- Write tests for new features

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

If you encounter any issues or have questions:
1. Check the [Issues](../../issues) section
2. Create a new issue with detailed information
3. Include logs and steps to reproduce the problem

## 🙏 Acknowledgments

- [Gin Web Framework](https://gin-gonic.com/) for the excellent HTTP framework
- [Firebase](https://firebase.google.com/) for backend services
- [Go](https://golang.org/) team for the amazing language
- Open source community for various packages used in this project 