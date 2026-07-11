import os
import zipfile
import psycopg2

# Database Configuration (Ensure this matches your target database)
DB_CONFIG = {
    "dbname": "your_database_name",
    "user": "your_username",
    "password": "your_password",
    "host": "localhost",
    "port": "5432"
}

# Change this to the exact name of the zip file you want to restore
ZIP_TO_RESTORE = "db_backup_20260711_120000.zip" 
TEMP_EXTRACT_DIR = "temp_restore"

def restore_database():
    if not os.path.exists(ZIP_TO_RESTORE):
        print(f"Error: Backup file '{ZIP_TO_RESTORE}' not found.")
        return

    try:
        # 1. Connect to the database
        print("Connecting to the database...")
        conn = psycopg2.connect(**DB_CONFIG)
        cursor = conn.cursor()

        # 2. Extract the ZIP file
        print(f"Extracting {ZIP_TO_RESTORE}...")
        os.makedirs(TEMP_EXTRACT_DIR, exist_ok=True)
        with zipfile.ZipFile(ZIP_TO_RESTORE, 'r') as zipf:
            zipf.extractall(TEMP_EXTRACT_DIR)

        # Get all extracted CSV files
        extracted_files = [f for f in os.listdir(TEMP_EXTRACT_DIR) if f.endswith('.csv')]

        # 3. Restore each table
        for filename in extracted_files:
            table_name = os.path.splitext(filename)[0]
            csv_path = os.path.join(TEMP_EXTRACT_DIR, filename)
            
            print(f"Restoring table: {table_name}...")
            
            # Read the header row to extract column names
            with open(csv_path, 'r', encoding='utf-8') as f:
                header = f.readline().strip()
            
            # Empty the existing table and restart identities (auto-increment keys)
            # RESTART IDENTITY ensures serial/bigserial columns reset properly
            # CASCADE handles tables with foreign key dependencies
            cursor.execute(f"TRUNCATE TABLE {table_name} RESTART IDENTITY CASCADE;")
            
            # Stream the CSV data back into the database
            with open(csv_path, 'r', encoding='utf-8') as f:
                query = f"COPY {table_name} ({header}) FROM STDIN WITH CSV HEADER"
                cursor.copy_expert(query, f)
        
        # Commit the changes to the database
        conn.commit()
        print("Database transaction committed.")

        # 4. Clean up extracted CSV files
        print("Cleaning up temporary files...")
        for filename in extracted_files:
            os.remove(os.path.join(TEMP_EXTRACT_DIR, filename))
        os.rmdir(TEMP_EXTRACT_DIR)

        print("Restore completed successfully!")

    except Exception as e:
        if 'conn' in locals():
            conn.rollback()
            print("Database transaction rolled back due to error.")
        print(f"An error occurred during restore: {e}")
    finally:
        if 'cursor' in locals(): cursor.close()
        if 'conn' in locals(): conn.close()

if __name__ == "__main__":
    restore_database()