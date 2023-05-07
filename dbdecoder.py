import sqlite3

# Connect to the SQLite database
conn = sqlite3.connect('mydatabase.db')

# Create a cursor object
c = conn.cursor()

# Execute a SELECT query to retrieve all the IP addresses from the users table
c.execute("SELECT ip_address FROM users")

# Fetch all the results and store them in a variable
results = c.fetchall()

# Print the results
for result in results:
    print(result[0])

# Close the database connection
conn.close()
