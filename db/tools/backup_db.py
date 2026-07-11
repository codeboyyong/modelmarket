import os
import csv
import zipfile
import psycopg2
from datetime import datetime

# Database Configuration
DB_CONFIG = {
    "dbname": "your_database_name",
    "user": "your_username",
    "password": "your_password",
    "host": "localhost",
    "port": "5432"
}

# Output Configuration
TIMESTAMP = datetime.now().strftime("%Y%m%d_%H%M%S")
ZIP_FILENAME = f"db_backup_{TIMESTAMP}.zip"
TEMP_DIR = "temp_backup"

def backup_database():
    try:
        # 1. Connect to the database
        print("Connecting to the database...")
        conn = psycopg2.connect(**DB_CONFIG)
        cursor = conn.cursor()

        # 2. Get all public table names
        cursor.execute("""
            SELECT table_name 
            FROM information_schema.tables 
            WHERE table_schema = 'public' AND table_type = 'BASE TABLE';
        """)
        tables = [row[0] for row in cursor.fetchall()]
        
        if not tables:
            print("No tables found to back up.")
            return

        # Create temporary directory for CSVs
        os.makedirs(TEMP_DIR, exist_ok=True)
        csv_files = []

        # 3. Export each table to CSV
        for table in tables:
            print(f"Exporting table: {table}...")
            csv_path = os.path.join(TEMP_DIR, f"{table}.csv")
            
            # Using PostgreSQL's COPY command for fast CSV export
            # We use standard Python file writing to handle the stream securely
            with open(csv_path, 'w', newline='', encoding='utf-8') as f:
                query = f"COPY {table} TO STDOUT WITH CSV HEADER"
                cursor.copy_expert(query, f)
            
            csv_files.append((csv_path, f"{table}.csv"))

        # 4. Zip all CSV files together
        print(f"Creating zip archive: {ZIP_FILENAME}...")
        with zipfile.ZipFile(ZIP_FILENAME, 'w', zipfile.ZIP_DEFLATED) as zipf:
            for file_path, arcname in csv_files:
                zipf.write(file_path, arcname)

        # 5. Clean up temporary CSV files
        print("Cleaning up temporary files...")
        for file_path, _ in csv_files:
            os.remove(file_path)
        os.rmdir(TEMP_DIR)

        print(f"Backup completed successfully! Saved as {ZIP_FILENAME}")

    except Exception as e:
        print(f"An error occurred during backup: {e}")
    finally:
        if 'cursor' in locals(): cursor.close()
        if 'conn' in locals(): conn.close()

if __name__ == "__main__":
    backup_database()