-- Create the Users table if it does not exist
CREATE TABLE IF NOT EXISTS Users ( 
    ID INTEGER PRIMARY KEY AUTOINCREMENT, 
    Name TEXT NOT NULL, 
    Email TEXT UNIQUE NOT NULL, 
    ContactNumber TEXT NOT NULL, 
    Password TEXT NOT NULL, 
    Role TEXT NOT NULL, 
    LibID INTEGER NOT NULL, 
    FOREIGN KEY (LibID) REFERENCES Library(ID) 
); 

-- Create the Library table if it does not exist
CREATE TABLE IF NOT EXISTS Library ( 
    ID INTEGER PRIMARY KEY AUTOINCREMENT, 
    Name TEXT UNIQUE NOT NULL 
); 

-- Create the BookInventory table if it does not exist
CREATE TABLE IF NOT EXISTS BookInventory ( 
    ISBN TEXT PRIMARY KEY, 
    LibID INTEGER NOT NULL, 
    Title TEXT NOT NULL, 
    Authors TEXT NOT NULL, 
    Publisher TEXT NOT NULL, 
    Version TEXT NOT NULL, 
    TotalCopies INTEGER NOT NULL, 
    AvailableCopies INTEGER NOT NULL, 
    FOREIGN KEY (LibID) REFERENCES Library(ID) 
); 

-- Create the RequestEvents table if it does not exist
CREATE TABLE IF NOT EXISTS RequestEvents ( 
    ReqID INTEGER PRIMARY KEY AUTOINCREMENT, 
    BookID TEXT NOT NULL, 
    ReaderID INTEGER NOT NULL, 
    RequestDate DATETIME NOT NULL, 
    ApprovalDate DATETIME, 
    ApproverID INTEGER, 
    RequestType TEXT NOT NULL, 
    FOREIGN KEY (BookID) REFERENCES BookInventory(ISBN), 
    FOREIGN KEY (ReaderID) REFERENCES Users(ID), 
    FOREIGN KEY (ApproverID) REFERENCES Users(ID) 
); 

-- Create the IssueRegistry table if it does not exist
CREATE TABLE IF NOT EXISTS IssueRegistry ( 
    IssueID INTEGER PRIMARY KEY AUTOINCREMENT, 
    ISBN TEXT NOT NULL, 
    ReaderID INTEGER NOT NULL, 
    IssueApproverID INTEGER NOT NULL, 
    IssueStatus TEXT NOT NULL, 
    IssueDate DATETIME NOT NULL, 
    ExpectedReturnDate DATETIME NOT NULL, 
    ReturnDate DATETIME, 
    ReturnApproverID INTEGER, 
    FOREIGN KEY (ISBN) REFERENCES BookInventory(ISBN), 
    FOREIGN KEY (ReaderID) REFERENCES Users(ID), 
    FOREIGN KEY (IssueApproverID) REFERENCES Users(ID), 
    FOREIGN KEY (ReturnApproverID) REFERENCES Users(ID) 
);