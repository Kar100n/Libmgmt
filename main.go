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

// CreateUser creates a new user
func CreateUser(db *sql.DB, name, email, contactNumber, password, role string, libID int) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	stmt, err := db.Prepare("INSERT INTO Users (Name, Email, ContactNumber, Password, Role, LibID) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(name, email, contactNumber, hashedPassword, role, libID)
	return err
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

	_, err = stmt.Exec(lib.Name)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, lib)
}

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

	err = CreateUser(db, user.Name, user.Email, user.ContactNumber, user.Password, user.Role, user.LibID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

func addBook(c *gin.Context) {
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

	stmt, err := db.Prepare("INSERT INTO BookInventory (ISBN, LibID, Title,Authors, Publisher, Version, TotalCopies, AvailableCopies) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(book.ISBN, book.LibID, book.Title, book.Authors, book.Publisher, book.Version, book.TotalCopies, book.TotalCopies)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, book)
}

func updateBook(c *gin.Context) {
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

	stmt, err := db.Prepare("UPDATE BookInventory SET Title = ?, Authors = ?, Publisher = ?, Version = ?, TotalCopies = ?, AvailableCopies = ? WHERE ISBN = ?")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(book.Title, book.Authors, book.Publisher, book.Version, book.TotalCopies, book.AvailableCopies, isbn)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, book)
}

func removeBook(c *gin.Context) {
	isbn := c.Param("isbn")

	db, err := sql.Open("sqlite3", "library.db")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer db.Close()

	stmt, err := db.Prepare("DELETE FROM BookInventory WHERE ISBN = ?")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(isbn)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Book removed successfully"})
}

func listIssueRequests(c *gin.Context) {
	db, err := sql.Open("sqlite3", "library.db")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT ReqID, BookID, ReaderID, RequestDate, ApprovalDate, ApproverID, RequestType FROM RequestEvents")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	var requests []RequestEvent
	for rows.Next() {
		var req RequestEvent
		if err := rows.Scan(&req.ReqID, &req.BookID, &req.ReaderID, &req.RequestDate, &req.ApprovalDate, &req.ApproverID, &req.RequestType); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		requests = append(requests, req)
	}

	c.JSON(http.StatusOK, requests)
}

func approveIssueRequest(c *gin.Context) {
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

	stmt, err := db.Prepare("UPDATE RequestEvents SET ApprovalDate = ?, ApproverID = ? WHERE ReqID = ?")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(time.Now(), req.ApproverID, reqID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if req.RequestType == "issue" {
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

		_, err = stmt.Exec(reqID, reqID, req.ApproverID, time.Now())
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Request updated successfully"})
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

func listAvailableBooks(c *gin.Context) {
	user, _ := c.Get("user")
	userObj := user.(User)

	db, err := sql.Open("sqlite3", "library.db")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT ISBN, Title, Authors, Publisher, Version, AvailableCopies FROM BookInventory WHERE LibID = ?", userObj.LibID)
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
