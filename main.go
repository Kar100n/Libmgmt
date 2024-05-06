package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/skip2/go-qrcode" // Importing go-qrcode package
	"golang.org/x/crypto/bcrypt"
)

const defaultOwnerEmail = "default_owner@example.com"
const defaultOwnerRole = "owner"

// User represents a user in the system
type User struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	ContactNumber string `json:"contact_number"`
	Role          string `json:"role"`
	LibID         int    `json:"lib_id"`
	Password      string `json:"-"`
}

// Library represents a library
type Library struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// BookInventory represents a book in the inventory
type BookInventory struct {
	ISBN            string `json:"isbn"`
	LibID           int    `json:"lib_id"`
	Title           string `json:"title"`
	Authors         string `json:"authors"`
	Publisher       string `json:"publisher"`
	Version         string `json:"version"`
	TotalCopies     int    `json:"total_copies"`
	AvailableCopies int    `json:"available_copies"`
}

// RequestEvent represents a book request event
type RequestEvent struct {
	ReqID        int       `json:"req_id"`
	BookID       string    `json:"book_id"`
	ReaderID     int       `json:"reader_id"`
	RequestDate  time.Time `json:"request_date"`
	ApprovalDate time.Time `json:"approval_date"`
	ApproverID   int       `json:"approver_id"`
	RequestType  string    `json:"request_type"`
}

// IssueRegistry represents an issued book
type IssueRegistry struct {
	IssueID            int       `json:"issue_id"`
	ISBN               string    `json:"isbn"`
	ReaderID           int       `json:"reader_id"`
	IssueApproverID    int       `json:"issue_approver_id"`
	IssueStatus        string    `json:"issue_status"`
	IssueDate          time.Time `json:"issue_date"`
	ExpectedReturnDate time.Time `json:"expected_return_date"`
	ReturnDate         time.Time `json:"return_date"`
	ReturnApproverID   int       `json:"return_approver_id"`
}

// createUser creates a new user
func createUser(c *gin.Context) {
	var user User
	if err := c.BindJSON(&user); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	db, err := sql.Open("sqlite3", "library.db")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer db.Close()

	// Check if the user is trying to create an owner for an existing library
	if user.Role == defaultOwnerRole {
		var existingOwnerID int
		err = db.QueryRow("SELECT ID FROM Users WHERE Role = ? AND LibID = ?", defaultOwnerRole, user.LibID).Scan(&existingOwnerID)
		if err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Owner already exists for this library"})
			return
		} else if err != sql.ErrNoRows {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user.Password = string(hashedPassword)

	stmt, err := db.Prepare("INSERT INTO users (name, email, ContactNumber, Password, role, LibID) VALUES (?,?,?,?,?,?)")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(user.Name, user.Email, user.ContactNumber, user.Password, user.Role, user.LibID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user.ID = int(id)

	c.JSON(http.StatusCreated, user)
}

// AuthMiddleware is a middleware for authentication
func AuthMiddleware(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		email, password, ok := c.Request.BasicAuth()
		if !ok {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		db, err := sql.Open("sqlite3", "library.db")
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		defer db.Close()

		var user User
		err = db.QueryRow("SELECT ID, Name, Email, ContactNumber, Role, LibID, Password FROM Users WHERE Email =?", email).Scan(&user.ID, &user.Name, &user.Email, &user.ContactNumber, &user.Role, &user.LibID, &user.Password)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		if user.Password != password {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		if !strings.EqualFold(user.Role, role) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Set("user", user)
		c.Next()
	}
}

func initDatabase() {
	db, err := sql.Open("sqlite3", "library.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	schemaBytes, err := os.ReadFile("Schema.sql")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(string(schemaBytes))
	if err != nil {
		log.Printf("%q: %s\n", err, string(schemaBytes))
		return
	}

	fmt.Println("Tables created successfully!")

	// Check if a default owner user exists
	var defaultUser User
	defaultUser.Email = defaultOwnerEmail
	defaultUser.Role = defaultOwnerRole
	defaultUser.LibID = 1
	err = db.QueryRow("SELECT ID, Name, ContactNumber FROM Users WHERE Email = ? AND Role = ? AND LibID = ?", defaultUser.Email, defaultUser.Role, defaultUser.LibID).Scan(&defaultUser.ID, &defaultUser.Name, &defaultUser.ContactNumber)
	if err != nil {
		fmt.Println("Error querying Users table:", err)
		// If not, create a default owner user
		defaultUser.Name = "Root"
		defaultUser.ContactNumber = "1234567890"
		defaultUser.Password = "password"
		_, err = db.Exec("INSERT INTO Users (Name, Email, ContactNumber, Password, Role, LibID) VALUES (?, ?, ?, ?, ?, ?)", defaultUser.Name, defaultUser.Email, defaultUser.ContactNumber, defaultUser.Password, defaultUser.Role, defaultUser.LibID)
		if err != nil {
			fmt.Println("Error inserting default owner user:", err)
			return
		}
	}
}

func main() {
	initDatabase()

	r := gin.Default()

	owner := r.Group("/owner", AuthMiddleware("owner"))
	{
		owner.POST("/libraries", createLibrary)
		owner.POST("/users", createUser)
	}

	admin := r.Group("/admin", AuthMiddleware("admin"))
	{
		admin.POST("/books", addBook)
		admin.POST("/users", createUser)
		admin.PUT("/books/:isbn", updateBook)
		admin.DELETE("/books/:isbn", removeBook)
		admin.GET("/requests", listIssueRequests)
		admin.POST("/requests/:reqID", approveIssueRequest)
		admin.GET("/readers/:readerID", getReaderInfo)
	}

	reader := r.Group("/reader", AuthMiddleware("reader"))
	{
		reader.POST("/requests", issueBookRequest)
		reader.GET("/books", listAvailableBooks)
	}

	r.Run(":8081")
}

func createLibrary(c *gin.Context) {
	var lib Library
	if err := c.BindJSON(&lib); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	db, err := sql.Open("sqlite3", "library.db")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer db.Close()

	stmt, err := db.Prepare("INSERT INTO Library (Name) VALUES (?)")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(lib.Name)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	id, _ := result.LastInsertId()
	lib.ID = int(id)
	c.JSON(http.StatusCreated, lib)
}

func CreateUser(c *gin.Context) {
	var user User
	if err := c.BindJSON(&user); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	db, err := sql.Open("sqlite3", "library.db")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer db.Close()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user.Password = string(hashedPassword)

	stmt, err := db.Prepare("INSERT INTO users (name, email, ContactNumber, Password, role, LibID) VALUES (?,?,?,?,?,?)")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(user.Name, user.Email, user.ContactNumber, user.Password, user.Role, user.LibID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user.ID = int(id)

	c.JSON(http.StatusCreated, user)
}

// addBook allows a library admin to add books to the inventory
func addBook(c *gin.Context) {
	user, _ := c.Get("user")
	userObj := user.(User)

	var book BookInventory
	if err := c.BindJSON(&book); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	db, err := sql.Open("sqlite3", "library.db")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer db.Close()

	// Check if the book already exists in the inventory
	var existingBookID int
	err = db.QueryRow("SELECT ISBN FROM BookInventory WHERE LibID = ? AND ISBN = ?", userObj.LibID, book.ISBN).Scan(&existingBookID)
	if err == nil {
		// Book already exists, increment the available copies
		_, err = db.Exec("UPDATE BookInventory SET TotalCopies = TotalCopies + ?, AvailableCopies = AvailableCopies + ? WHERE ID = ?", book.TotalCopies, book.TotalCopies, existingBookID)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Book already exists, copies incremented"})
		return
	} else if err != sql.ErrNoRows {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Book doesn't exist, insert a new record
	stmt, err := db.Prepare("INSERT INTO BookInventory (ISBN, LibID, Title, Authors, Publisher, Version, TotalCopies, AvailableCopies) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(book.ISBN, userObj.LibID, book.Title, book.Authors, book.Publisher, book.Version, book.TotalCopies, book.TotalCopies)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Generate QR code with book details
	qrContent := fmt.Sprintf("Title: %s\nAuthor: %s\nISBN: %s\nVersion: %s", book.Title, book.Authors, book.ISBN, book.Version)
	qrFilename := fmt.Sprintf("qr_%s.png", book.ISBN) // Naming QR code file with ISBN
	qrFilePath := "./qr_codes/" + qrFilename          // Path to store the QR code file
	err = qrcode.WriteFile(qrContent, qrcode.Medium, 256, qrFilePath)
	if err != nil {
		log.Println("Error generating QR code:", err)
		// Handle error if needed
	}

	c.JSON(http.StatusOK, gin.H{"message": "Book added successfully", "qr_code": qrFilePath})
}

// updateBook allows a library admin to update the details of a book
func updateBook(c *gin.Context) {
	user, _ := c.Get("user")
	userObj := user.(User)

	isbn := c.Param("isbn")

	var book BookInventory
	if err := c.BindJSON(&book); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	db, err := sql.Open("sqlite3", "library.db")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer db.Close()

	// Check if the book exists in the inventory
	var existingBookID int
	err = db.QueryRow("SELECT ID FROM BookInventory WHERE LibID = ? AND ISBN = ?", userObj.LibID, isbn).Scan(&existingBookID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Book not found in the inventory"})
		} else {
			c.AbortWithError(http.StatusInternalServerError, err)
		}
		return
	}

	// Update the book details
	stmt, err := db.Prepare("UPDATE BookInventory SET Title = ?, Authors = ?, Publisher = ?, Version = ?, TotalCopies = ? WHERE ID = ?")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(book.Title, book.Authors, book.Publisher, book.Version, book.TotalCopies, existingBookID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Book details updated successfully"})
}

// removeBook allows a library admin to remove a book from the inventory
func removeBook(c *gin.Context) {
	user, _ := c.Get("user")
	userObj := user.(User)

	isbn := c.Param("isbn")

	db, err := sql.Open("sqlite3", "library.db")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer db.Close()

	// Check if the book exists in the inventory
	var existingBookID int
	err = db.QueryRow("SELECT ID FROM BookInventory WHERE LibID = ? AND ISBN = ?", userObj.LibID, isbn).Scan(&existingBookID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Book not found in the inventory"})
		} else {
			c.AbortWithError(http.StatusInternalServerError, err)
		}
		return
	}

	// Check if there are any issued copies of the book
	var issuedCopies int
	err = db.QueryRow("SELECT COUNT(*) FROM IssueRegistry WHERE ISBN = ? AND IssueStatus = 'issued'", isbn).Scan(&issuedCopies)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if issuedCopies > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot remove book with issued copies"})
		return
	}

	// Remove the book from the inventory
	stmt, err := db.Prepare("DELETE FROM BookInventory WHERE ID = ?")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(existingBookID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Book removed successfully"})
}

// listIssueRequests allows a library admin to list issue requests in their library
func listIssueRequests(c *gin.Context) {
	user, _ := c.Get("user")
	userObj := user.(User)

	db, err := sql.Open("sqlite3", "library.db")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT ReqID, BookID, ReaderID, RequestDate, ApprovalDate, ApproverID, RequestType FROM RequestEvents WHERE ApproverID = ? AND RequestType = 'issue'", userObj.ID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	var requests []RequestEvent
	for rows.Next() {
		var req RequestEvent
		var approvalDate sql.NullTime
		var approverID sql.NullInt64
		if err := rows.Scan(&req.ReqID, &req.BookID, &req.ReaderID, &req.RequestDate, &approvalDate, &approverID, &req.RequestType); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		if approvalDate.Valid {
			req.ApprovalDate = approvalDate.Time
		} else {
			req.ApprovalDate = time.Time{}
		}

		if approverID.Valid {
			req.ApproverID = int(approverID.Int64)
		} else {
			req.ApproverID = 0
		}

		requests = append(requests, req)
	}

	c.JSON(http.StatusOK, requests)
}

// approveIssueRequest allows a library admin to approve or reject an issue request
func approveIssueRequest(c *gin.Context) {
	user, _ := c.Get("user")
	userObj := user.(User)

	reqID := c.Param("reqID")

	var req RequestEvent
	if err := c.BindJSON(&req); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	db, err := sql.Open("sqlite3", "library.db")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer db.Close()

	// Check if the request exists and if it's pending approval
	var existingRequestID int
	err = db.QueryRow("SELECT ReqID FROM RequestEvents WHERE ReqID = ? AND ApproverID IS NULL AND RequestType = 'issue'", reqID).Scan(&existingRequestID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Issue request not found or already approved/rejected"})
		} else {
			c.AbortWithError(http.StatusInternalServerError, err)
		}
		return
	}

	// Update the request status with approver ID and approval date
	stmt, err := db.Prepare("UPDATE RequestEvents SET ApproverID = ?, ApprovalDate = ? WHERE ReqID = ?")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(userObj.ID, time.Now(), reqID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if req.RequestType == "issue" {
		// Update book inventory and issue registry if the request is approved
		stmt, err := db.Prepare("UPDATE BookInventory SET AvailableCopies = AvailableCopies - 1 WHERE ISBN = (SELECT BookID FROM RequestEvents WHERE ReqID = ?)")
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		defer stmt.Close()

		_, err = stmt.Exec(reqID)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		stmt, err = db.Prepare("INSERT INTO IssueRegistry (ISBN, ReaderID, IssueApproverID, IssueStatus, IssueDate, ExpectedReturnDate) VALUES ((SELECT BookID FROM RequestEvents WHERE ReqID = ?), (SELECT ReaderID FROM RequestEvents WHERE ReqID = ?), ?, 'issued', ?, DATE('now', '+30 days'))")
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		defer stmt.Close()

		_, err = stmt.Exec(reqID, reqID, userObj.ID, time.Now())
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Issue request updated successfully"})
}

func getReaderInfo(c *gin.Context) {
	readerID := c.Param("readerID")

	db, err := sql.Open("sqlite3", "library.db")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer db.Close()

	var user User
	err = db.QueryRow("SELECT ID, Name, Email, ContactNumber, Role, LibID FROM Users WHERE ID = ?", readerID).Scan(&user.ID, &user.Name, &user.Email, &user.ContactNumber, &user.Role, &user.LibID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// issueBookRequest allows a reader to raise an issue request
func issueBookRequest(c *gin.Context) {
	user, _ := c.Get("user")
	userObj := user.(User)

	var req RequestEvent
	if err := c.BindJSON(&req); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	req.ReaderID = userObj.ID
	req.RequestType = "issue"
	req.RequestDate = time.Now()

	db, err := sql.Open("sqlite3", "library.db")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer db.Close()

	// Check if the requested book is available
	var availableCopies int
	err = db.QueryRow("SELECT AvailableCopies FROM BookInventory WHERE LibID = ? AND ISBN = ? AND AvailableCopies > 0", userObj.LibID, req.BookID).Scan(&availableCopies)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Requested book is not available"})
		return
	}

	stmt, err := db.Prepare("INSERT INTO RequestEvents (BookID, ReaderID, RequestDate, RequestType) VALUES (?, ?, ?, ?)")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(req.BookID, req.ReaderID, req.RequestDate, req.RequestType)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, req)
}

// listAvailableBooks lists available books for the reader's library
func listAvailableBooks(c *gin.Context) {
	user, _ := c.Get("user")
	userObj := user.(User)

	db, err := sql.Open("sqlite3", "library.db")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT ISBN, Title, Authors, Publisher, Version, AvailableCopies FROM BookInventory WHERE LibID = ? AND AvailableCopies > 0", userObj.LibID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	var books []BookInventory
	for rows.Next() {
		var book BookInventory
		if err := rows.Scan(&book.ISBN, &book.Title, &book.Authors, &book.Publisher, &book.Version, &book.AvailableCopies); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		books = append(books, book)
	}

	c.JSON(http.StatusOK, books)
}
