Here are all the curl commands for the functions in the provided Go code:

1. Create a new library (for owner role):
```
curl -X POST -H "Content-Type: application/json" -d '{"name":"Library Name"}' -u owner@example.com:password http://localhost:8081/owner/libraries
```

2. Create a new user (for owner role):
```
curl -X POST -H "Content-Type: application/json" -d '{"name":"User Name","email":"user@example.com","contact_number":"1234567890","password":"password","role":"reader","lib_id":1}' -u owner@example.com:password http://localhost:8081/owner/users
```

3. Add a new book (for admin role):
```
curl -X POST -H "Content-Type: application/json" -d '{"isbn":"978-3-16-148410-0","lib_id":1,"title":"Book Title","authors":"Author Name","publisher":"Publisher Name","version":"1.0","total_copies":10,"available_copies":10}' -u admin@example.com:password http://localhost:8081/admin/books
```

4. Update a book (for admin role):
```
curl -X PUT -H "Content-Type: application/json" -d '{"title":"Updated Title","authors":"Updated Author","publisher":"Updated Publisher","version":"2.0","total_copies":15,"available_copies":15}' -u admin@example.com:password http://localhost:8081/admin/books/978-3-16-148410-0
```

5. Remove a book (for admin role):
```
curl -X DELETE -u admin@example.com:password http://localhost:8081/admin/books/978-3-16-148410-0
```

6. List issue requests (for admin role):
```
curl -u admin@example.com:password http://localhost:8081/admin/requests
```

7. Approve an issue request (for admin role):
```
curl -X POST -H "Content-Type: application/json" -d '{"approver_id":1}' -u admin@example.com:password http://localhost:8081/admin/requests/1
```

8. Get reader information (for admin role):
```
curl -u admin@example.com:password http://localhost:8081/admin/readers/1
```

9. Issue a book request (for reader role):
```
curl -X POST -H "Content-Type: application/json" -d '{"book_id":"978-3-16-148410-0"}' -u reader@example.com:password http://localhost:8081/reader/requests
```

10. List available books (for reader role):
```
curl -u reader@example.com:password http://localhost:8081/reader/books
```

Note: Replace the placeholders (`owner@example.com`, `admin@example.com`, `reader@example.com`, `password`, and other values) with the appropriate values for your setup.