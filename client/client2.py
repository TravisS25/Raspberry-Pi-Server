#/usr/bin/python3.5
import time
import requests
from datetime import datetime
import os
from shutil import copyfile
import sqlite3
import sys
import shutil

# Name of device which will be used when sending to server
DEVICE_NAME = "Device2"

# Csv file that will be used to write the timestamps of the motions
CSV_FILE = os.path.join("csv", DEVICE_NAME + ".csv")

# File to keep things up-to-date if the device is turned off
# File will only have one line, comma seperated and will be associated
# with variable values below with order:
#   1. is_recording
#   2. has_internet
#   3. had_internet_before
#   4. has_new_set_not_recording
#   5. current_set
# STATUS_FILE = DEVICE_NAME + "_status.txt"

# Filepath to directory where sets of csv files will be stored 
# when requesting a new set from server
SETS_DIRECTORY = "csv/sets"

DATABASE_FILE = DEVICE_NAME + ".db"
CREATE_TABLE = "CREATE TABLE IF NOT EXISTS `device_status` ( " +\
	"`pk`						 INTEGER PRIMARY KEY AUTOINCREMENT, " +\
	"`has_internet`  			 INTEGER NOT NULL, " +\
	"`had_internet_before`		 INTEGER NOT NULL, " +\
	"`is_recording`  			 INTEGER NOT NULL, " +\
	"`has_new_set_not_recording` INTEGER NOT NULL, " +\
	"`device_set`				 INTEGER NOT NULL " +\
");"
DEFAULT_INSERT_STATEMENT = \
"INSERT INTO device_status (has_internet, had_internet_before, is_recording, has_new_set_not_recording, device_set) " +\
"VALUES (1, 1, 1, 0, 1);"
QUERY_STATEMENT =  "SELECT has_internet, had_internet_before, is_recording, has_new_set_not_recording, device_set FROM device_status WHERE pk=1"

def modify_status_file(insert, index):
    """
    Helper function to modify status file
    """
    new_line = ""
    with open(STATUS_FILE, 'r+') as f:
        line_array = f.readline().split(",")
        line_array[index] = insert

        for item in line_array:
            new_line = new_line + str(item) + ","
        
        f.write(new_line)

def init():
    is_recording = True
    has_internet = True
    had_internet_before = True
    has_new_set_not_recording = False
    current_set = 1

    if not os.path.isdir(SETS_DIRECTORY):
        os.makedirs(SETS_DIRECTORY)

    if not os.path.exists(CSV_FILE):
        with open(CSV_FILE, 'w') as f:
            pass

    # if not os.path.exists(STATUS_FILE):
    #     with open(STATUS_FILE, 'w') as f:
    #         f.write("True,True,True,True,1")
    #         is_recording = True
    #         has_internet = True
    #         had_internet_before = True
    #         has_new_set_not_recording = False
    #         current_set = 1
    # else:
    #     with open(STATUS_FILE, 'r') as f:
    #         line_array = f.readline().split(",")

    #         for i, item in enumerate(line_array):
    #             if i == 0:
    #                 is_recording = bool(item)
    #             elif i == 1:
    #                 has_internet = bool(item)
    #             elif i == 2:
    #                 had_internet_before = bool(item)
    #             elif i == 3:
    #                 has_new_set_not_recording = bool(item)
    #             else:
    #                 current_set = int(item)
            
    if not os.path.exists(DATABASE_FILE):
        with open(DATABASE_FILE, 'w') as f:
            pass

    try:
        conn = sqlite3.connect(DATABASE_FILE)
        c = conn.cursor()
        c.execute(CREATE_TABLE)
        conn.commit()

        c.execute(QUERY_STATEMENT)
        row = c.fetchone()

        if not row:
            c.execute(DEFAULT_INSERT_STATEMENT)
            conn.commit()
        else:
            has_internet = row[0]
            had_internet_before = row[1]
            is_recording = row[2]
            has_new_set_not_recording = row[3]
            current_set = row[4]

            print(row)

        c.close()

    except Exception as e:
        print(e)
        return

    while True:
        time.sleep(3)
        payload = {"password": "test"}

        if is_recording:
            has_new_set_not_recording = False
            # conn.execute("UPDATE device_status SET (has_new_set_not_recording = ?) WHERE pk=1;", (0))
            # conn.commit()
            # modify_status_file(has_new_set_not_recording, 3)

            if has_internet and not had_internet_before:
                print("reloading csv file")
                with open(CSV_FILE, 'rb') as f:
                    # f.read()
                    payload.update({"fileName": DEVICE_NAME + ".csv"})
                    reload_url = "http://localhost:8003/reload-csv/"
                    print(str(payload))
                    r = requests.post(
                        reload_url, 
                        payload,
                        files={"uploadFile": f}
                    )
                had_internet_before = True
                conn.execute("UPDATE device_status SET had_internet_before=? WHERE pk=1", ('1'))
                conn.commit()
                # modify_status_file(had_internet_before, 2)

            current_date = datetime.now().strftime("%Y-%m-%d")
            current_time = datetime.now().strftime("%H:%M:%S")
            movement = 1
            time_stamp = DEVICE_NAME + "," + \
                         current_date + "," + \
                         current_time + "," + \
                         str(movement) + \
                         "\n"
            
            with open(CSV_FILE, 'a') as f:
                f.write(time_stamp)

            sensor_url = "http://localhost:8003/sensor-handler/"
            payload.update({"timeStamp": time_stamp})
        
            try:
                r = requests.post(sensor_url, payload)
                print("Sending to server...")
                response = str(r._content.decode("utf-8")).split(",")

                for item in response:
                    if item == "Stop Recording":
                        is_recording = False
                        # modify_status_file(False, 0)
                        conn.execute("UPDATE device_status SET is_recording=? WHERE pk=1", ('0'))
                        conn.commit()
                    if item == "New Set":
                        set_csv_file = os.path.join(SETS_DIRECTORY, DEVICE_NAME, str(current_set) + ".csv")
                        current_set = current_set + 1
                        # modify_status_file(current_set, 4)
                        conn.execute("UPDATE device_status SET device_set=? WHERE pk=1", (str(current_set)) )
                        conn.commit()
                        copyfile(CSV_FILE, set_csv_file)
                        csv_file = open(CSV_FILE, 'w')
                        csv_file.close()

                has_internet = True
                conn.execute("UPDATE device_status SET has_internet=? WHERE pk=1", ('1'))
                conn.commit()
                # modify_status_file(has_internet, 1)
            except Exception as e:
                # print(e)
                print("Have no internet but still going...")
                has_internet = False
                had_internet_before = False
                conn.execute("UPDATE device_status SET has_internet=? WHERE pk=1", ('0'))
                conn.execute("UPDATE device_status SET had_internet_before=? WHERE pk=1", ('0'))
                conn.commit()
                # modify_status_file(has_internet, 1)
                # modify_status_file(had_internet_before, 2)
        else:
            try:
                r = requests.get(
                    "http://localhost:8003/device-status-handler/",
                     params={"deviceName": DEVICE_NAME}
                )
                print("Not recording but still going...")
                response = str(r._content.decode("utf-8")).split(",")

                for item in response:
                    if item == "Record":
                        is_recording = True
                        conn.execute("UPDATE device_status SET is_recording=? WHERE pk=1", ('1'))
                        conn.commit()
                        # modify_status_file(is_recording, 0)
                    if item == "New Set" and not has_new_set_not_recording:
                        print("new set while not recording")
                        set_csv_file = os.path.join(SETS_DIRECTORY, DEVICE_NAME, str(current_set) + ".csv")
                        current_set = current_set + 1
                        has_new_set_not_recording = True
                        conn.execute("UPDATE device_status SET device_set=? WHERE pk=1", (str(current_set)))
                        conn.execute("UPDATE device_status SET has_new_set_not_recording=? WHERE pk=1", ('1'))
                        conn.commit()
                        # modify_status_file(current_set, 4)
                        # modify_status_file(has_new_set_not_recording, 3)
                        copyfile(CSV_FILE, set_csv_file)
                        csv_file = open(CSV_FILE, 'w')
                        csv_file.close()

            except Exception as e:
                print(e)
                has_internet = False
                had_internet_before = False
                conn.execute("UPDATE device_status SET has_internet=? WHERE pk=1", ('0'))
                conn.execute("UPDATE device_status SET had_internet_before=? WHERE pk=1", ('0'))
                conn.commit()
                # modify_status_file(has_internet, 1)
                # modify_status_file(had_internet_before, 2)
                print("Not recording and no internet but still going...")


if __name__ == '__main__':
    if len(sys.argv) > 1:
        if sys.argv[1] == "wipe":
            os.remove(DEVICE_NAME + ".db")
            shutil.rmtree("csv")

    init()


                