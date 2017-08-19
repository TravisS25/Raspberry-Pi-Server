# Filepath to directory where sets of csv files will be stored 
# when requesting a new set from server
SETS_DIRECTORY = "csv/sets"
CSV_FILE = "csv/client.csv"

CREATE_TABLE = "CREATE TABLE IF NOT EXISTS `device_status` ( " +\
	"`device_name`  			 TEXT PRIMARY KEY, " +\
	"`has_internet`  			 INTEGER NOT NULL, " +\
	"`had_internet_before`		 INTEGER NOT NULL, " +\
	"`is_recording`  			 INTEGER NOT NULL, " +\
	"`has_new_set_not_recording` INTEGER NOT NULL, " +\
	"`device_set`				 INTEGER NOT NULL " +\
");"
DEFAULT_INSERT_STATEMENT = \
"INSERT INTO device_status (device_name, has_internet, had_internet_before, is_recording, has_new_set_not_recording, device_set) " +\
"VALUES (?, 1, 1, 1, 0, 1);"
QUERY_STATEMENT =  "SELECT device_name, has_internet, had_internet_before, is_recording, has_new_set_not_recording, device_set FROM device_status WHERE pk=1"
DATABASE_FILE = "client.db"