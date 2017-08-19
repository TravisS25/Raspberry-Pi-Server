import os


class Device():
    # is_recording = True
    # has_internet = True
    # had_internet_before = True
    # has_new_set_not_recording = False
    # current_set = 1

    def __init__(self, device_name="Client", ip_address="localhost:8003", sleep=.5, is_recording=True,
    has_internet=True, had_internet_before=True, has_new_set_not_recording=True, current_set=1, password="test",
    https=True, is_checked_in=False):
        self.device_name = device_name
        self.ip_address = ip_address
        self.sleep = sleep
        self.csv_file = "csv/" + self.device_name + ".csv"
        self.is_recording = is_recording
        self.has_internet = has_internet
        self.had_internet_before = had_internet_before
        self.has_new_set_not_recording = has_new_set_not_recording
        self.current_set = current_set
        self.password = password
        self.is_checked_in=is_checked_in

        if https:
            self.protocol = "https://"
        else:
            self.protocol = "http://"
        