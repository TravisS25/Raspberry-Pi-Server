import time
import requests
from datetime import datetime
import os
import sqlite3
from client import DEVICE_NAME

if __name__ == '__main__':
    try:
        conn = sqlite3.connect(DEVICE_NAME + '.db')
        c = conn.cursor()

    except Exception as e:
        print(e)
        return

    c.execute(
        "SELECT device_recording.device_set " +
        "FROM device_recording " +
        "ORDER BY device_recording.device_set DESC"
    )
    row = c.fetchone

    if row:
        current_set = row[0]
    else:
        current_set = 1

    for i in range(1, current_set + 1)
        c.execute(
            "SELECT dr.device_name, date, time, movement" +
            "FROM device_recording AS dr " +
        )

