#/usr/bin/python3.5
import time
import requests
from datetime import datetime
import os
from shutil import copyfile
import sqlite3
import sys
import shutil
import device
import const
import getopt
import configparser
import random


def _check_in_device(pi_device, config):
    payload = {"password": pi_device.password, "deviceName": pi_device.device_name}
    check_in_url = pi_device.protocol + pi_device.ip_address + "/check-in-handler/"
    try:
        r = requests.post(check_in_url, data=payload)
        result = r._content.decode('UTF-8')

        if result == "Already checked in":
            config["device"]["is_checked_in"] = "False"
            print("Device name is already in use, not sending to server but still running locally...")
        else:
            pi_device.is_checked_in = True
            config["device"]["is_checked_in"] = "True"
    except Exception as e:
        print("Have no internet but still going...")
        pi_device.has_internet = False
        pi_device.had_internet_before = False
        config["device"]["has_internet"] = "False"
        config["device"]["had_internet_before"] = "False"

        with open("client.ini", "w+") as configFile:
            config.write(configFile)
    

def _get_pi_device(config):
    check_file_message = "Check client.ini file or documentation for more"

    error = False
    device_name = config["DEFAULT"]["device_name"]
    ip_address = config["DEFAULT"]["ip_address"]
    sleep = config["DEFAULT"]["sleep"]
    https = config["DEFAULT"]["https"]
    has_internet = config["device"]["has_internet"]
    had_internet_before = config["device"]["had_internet_before"]
    has_new_set_not_recording = config["device"]["has_new_set_not_recording"]
    is_recording = config["device"]["is_recording"]
    device_set = config["device"]["device_set"]
    password = config["device"]["password"]

    if not device_name:
        print("device_name needs to be given.  " + check_file_message)
        error = True
    if not ip_address:
        print("ip_address needs to be given.  " + check_file_message)
        error = True
    if not sleep:
        print("sleep needs to be given.  " + check_file_message)
        error = True
    try:
        sleep = float(sleep)
    except ValueError:
        print("sleep must be an float.  " + check_file_message)
        error = True
    if float(sleep) < 0:
        print("sleep must be greater than 0.  " + check_file_message)
        error = True
    if not has_internet:
        print("has_internet settings is needed.  " + check_file_message)
        error = True
    try: 
        has_internet = bool(has_internet)
    except ValueError as e:
        print("has_internet should be boolean.  " + check_file_message)
        error = True
    if not had_internet_before:
        print("had_internet_before settings is needed.  " + check_file_message)
        error = True
    try:
        had_internet_before =  bool(had_internet_before)
    except ValueError as e:
        print("had_internet_before should be boolean.  " + check_file_message)
        error = True
    if not has_new_set_not_recording:
        print("has_new_set_not_recording setting is needed.  " + check_file_message)
        error = True
    try:
        has_new_set_not_recording = bool(has_new_set_not_recording)
    except ValueError as e:
        print("has_new_set_not_recording should be boolean.  " + check_file_message)
        error = True
    if not is_recording:
        print("is_recording setting is needed.  " + check_file_message)
        error = True
    try:
        is_recording = bool(is_recording)
    except ValueError as e:
        print("is_recording should be boolean.  " + check_file_message)
        error = True
    if not device_set:
        print("device_set setting is needed.  " + check_file_message)
        error = True
    try:
        device_set = int(device_set)

        if device_set < 1:
            print("device_set should be greater than 0.  " + check_file_message)
            error = True
    except ValueError as e:
        print("device_set should be int.  " + check_file_message)
        error = True
    if not password:
        print("password should be set.  " + check_file_message)
        error = True
    if len(password) < 5:
        print("password has to be at least 6 characters long.  " + check_file_message)
        error = True
    try:
        https = https.lower()
        if https == "true":
            https = True
        elif https == "false":
            https = False
        else:
            print("https setting needs to be bool")
            error = True
    except ValueError as e:
        print("https should be boolean.  " + check_file_message)
        error = True

    if error:
        print("Stopping device...")
        sys.exit(2)

    pi_device = device.Device(
        device_name=device_name,
        ip_address=ip_address,
        sleep=sleep, 
        is_recording=is_recording,
        has_internet=has_internet,
        had_internet_before=had_internet_before,
        has_new_set_not_recording=has_new_set_not_recording,
        current_set=device_set,
        password=password,
        https=https
    )
    return pi_device


def _init_file_system(pi_device):
    if not os.path.isdir(os.path.join(const.SETS_DIRECTORY, pi_device.device_name)):
        os.makedirs(os.path.join(const.SETS_DIRECTORY, pi_device.device_name))

    if not os.path.exists(pi_device.csv_file):
        with open(pi_device.csv_file, 'w') as f:
            pass


def init(pi_device, config):
    while True:
        time.sleep(pi_device.sleep)
        payload = {"password": pi_device.password}

        if pi_device.is_recording:
            pi_device.has_new_set_not_recording = False
            config["device"]["is_recording"] = "True"
            config["device"]["has_new_set_not_recording"] = "False"

            if pi_device.has_internet and not pi_device.had_internet_before:
                print("reloading csv file")
                with open(pi_device.csv_file, 'rb') as f:
                    payload.update({"fileName": pi_device.device_name + ".csv"})
                    reload_url = "http://"+pi_device.ip_address+"/reload-csv/"
                    print(str(payload))
                    r = requests.post(
                        reload_url, 
                        payload,
                        files={"uploadFile": f}
                    )

                pi_device.had_internet_before = True
                config["device"]["had_internet_before"] = "True"


            # Movement will be replaced by motion sensor code that indicates
            # if there had been movement or not
            movement = random.getrandbits(1)
            print("movement " + str(movement))
            current_date = datetime.now().strftime("%Y-%m-%d")
            current_time = datetime.now().strftime("%H:%M:%S")
            time_stamp = pi_device.device_name + "," + \
                            current_date + "," + \
                            current_time + "," + \
                            str(movement) + \
                            "\n"

            if movement == 1:
                with open(pi_device.csv_file, 'a') as f:
                    f.write(time_stamp)
                    print("Writing to file...")

            sensor_url = "http://"+pi_device.ip_address+"/sensor-handler/"
            payload.update({"timeStamp": time_stamp})
        
            try:
                if pi_device.is_checked_in:
                    r = requests.post(sensor_url, payload)
                    print("Sending to server...")
                    response = str(r._content.decode("utf-8")).split(",")

                    for item in response:
                        if item == "Stop Recording":
                            pi_device.is_recording = False
                            config["device"]["is_recording"] = "False"
                        if item == "New Set":
                            set_csv_file = os.path.join(const.SETS_DIRECTORY, pi_device.device_name, str(pi_device.current_set) + ".csv")
                            pi_device.current_set = pi_device.current_set + 1
                            config["device"]["device_set"] = pi_device.current_set
                            copyfile(pi_device.csv_file, set_csv_file)
                            csv_file = open(CSV_FILE, 'w')
                            csv_file.close()
                        if item == "Wrong Password":
                            print(item + ", not writing to server but still locally...")
                        if item == "Device does not exist":
                            print(item + " on server...")

                    pi_device.has_internet = True
                    config["device"]["has_internet"] = "True"
                else:
                    _check_in_device(pi_device)
            except Exception as e:
                print("Have no internet but still going...")
                pi_device.has_internet = False
                pi_device.had_internet_before = False
                config["device"]["has_internet"] = "False"
                config["device"]["had_internet_before"] = "False"

            with open("client.ini", "w+") as configFile:
                config.write(configFile)
        else:
            config["device"]["is_recording"] = "False"

            try:
                r = requests.get(
                    pi_device.protocol+pi_device.ip_address+"/device-status-handler/",
                    params={"deviceName": pi_device.device_name}
                )
                print("Not recording but still going...")
                response = str(r._content.decode("utf-8")).split(",")

                for item in response:
                    if item == "Record":
                        pi_device.is_recording = True
                        config["device"]["is_recording"] = "True"
                    if item == "New Set" and not pi_device.has_new_set_not_recording:
                        print("new set while not recording")
                        set_csv_file = os.path.join(const.SETS_DIRECTORY, pi_device.device_name, str(pi_device.current_set) + ".csv")
                        pi_device.current_set = pi_device.current_set + 1
                        config["device"]["device_set"] = pi_device.current_set
                        pi_device.has_new_set_not_recording = True
                        config["device"]["has_new_set_not_recording"] = "True"
                    
                        copyfile(pi_device.csv_file, set_csv_file)
                        csv_file = open(pi_device.csv_file, 'w')
                        csv_file.close()

            except Exception as e:
                pi_device.has_internet = False
                pi_device.had_internet_before = False
                config["device"]["has_internet"]
                config["device"]["had_internet_before"]
                print("Not recording and no internet but still going...")

            with open("client.ini", "w+") as configFile:
                config.write(configFile)


if __name__ == '__main__':
    config = configparser.ConfigParser()
    config.read("client.ini")
    pi_device = _get_pi_device(config)

    try:
        opts, args = getopt.getopt(sys.argv[1:],"w",["wipe"])
    except getopt.GetoptError:
      print('client.py -w --wipe')
      sys.exit(2)

    for opt, arg in opts:
        if opt in ("-w", "--wipe"):
            answer = input("You are about to delete current and all csv files.  Are you sure you want to continue? (y/n)")
            if answer == "y" or answer == "Y":
                print("Client wiped")
                if os.path.exists(os.path.join("csv", pi_device.device_name + ".csv")):
                    os.remove(os.path.join("csv", pi_device.device_name + ".csv"))
                    shutil.rmtree(os.path.join("csv", "sets", pi_device.device_name))
                else:
                    print("Nothing was deleted")

    if not os.path.exists("client.ini"):
        print("Must have client.ini file in root directory")
        print("Stopping device...")
        sys.exit(2)

    _init_file_system(pi_device)
    _check_in_device(pi_device, config)
    init(pi_device, config)


                