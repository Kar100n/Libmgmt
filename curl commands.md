Sure, here are the cURL commands to test the full functionality of the provided Go code:

1. **Create a Library**
```bash
curl -X POST -H "Content-Type: application/json" -d '{"name":"MyLibrary"}' http://localhost:8080/admin/libraries
```

2. **Add a User**
```bash
curl -X POST -H "Content-Type: application/json" -d '{"name":"John Doe", "email":"john@example.com", "contact_number":"1234567890", "role":"Reader", "library_id":1}' http://localhost:8080/admin/users
```

3. **Add a Book**
```bash
curl -X POST -H "Content-Type: application/json" -d '{"lib_id":1, "isbn":"978-3-16-148410-0", "title":"Book Title", "authors":"John Smith", "publisher":"Publisher Name", "version":"1.0", "total_copies":10, "available_copies":10}' http://localhost:8080/admin/books
```

4. **Remove a Book**
```bash
curl -X DELETE -H "Content-Type: application/json" -d '{"book_id":1}' http://localhost:8080/admin/books/978-3-16-148410-0
```

5. **Update a Book**
```bash
curl -X PUT -H "Content-Type: application/json" -d '{"isbn":"978-3-16-148410-0", "title":"Updated Book Title", "authors":"Updated Author", "publisher":"Updated Publisher", "version":"2.0"}' http://localhost:8080/admin/books/978-3-16-148410-0
```

6. **List Issue Requests**
```bash
curl -X GET http://localhost:8080/admin/issue-requests
```

7. **Approve or Reject an Issue Request**
```bash
curl -X PUT -H "Content-Type: application/json" -d '{"request_id":1, "approved":true, "approver_id":1, "approval_date":"2023-05-08T12:00:00Z"}' http://localhost:8080/admin/issue-requests/1
```

8. **Search for a Book**
```bash
curl -X POST -H "Content-Type: application/json" -d '{"title":"Book", "author":"John Smith", "publisher":"Publisher Name"}' http://localhost:8080/reader/search
```

9. **Raise an Issue Request**
```bash
curl -X POST -H "Content-Type: application/json" -d '{"book_id":1, "email":"john@example.com"}' http://localhost:8080/reader/issue-requests
```

Please note that you may need to adjust the values in the JSON payloads (e.g., book ID, user ID, library ID) according to the data in your database.