BEGIN TRANSACTION;
CREATE TABLE IF NOT EXISTS `device_status` (
	`pk`					INTEGER PRIMARY KEY AUTOINCREMENT,
	`device_name`			TEXT NOT NULL UNIQUE,
	`device_set`			INTEGER NOT NULL,
	`device_time`			TEXT NOT NULL,
	`is_new_set`			INTEGER NOT NULL,
	`is_recording`			INTEGER NOT NULL,
	`is_device_checked_in`  INTEGER NOT NULL
);
COMMIT;
