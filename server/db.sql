BEGIN TRANSACTION;
CREATE TABLE IF NOT EXISTS `device_status` (
	`pk`	INTEGER PRIMARY KEY AUTOINCREMENT,
	`device_name`	TEXT NOT NULL UNIQUE,
	`new_set`	INTEGER NOT NULL,
	`recording`	INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS "device_recording" (
	`pk`	INTEGER PRIMARY KEY AUTOINCREMENT,
	`device_name`	TEXT NOT NULL,
	`date`	TEXT NOT NULL,
	`time`	TEXT NOT NULL,
	`movement`	INTEGER NOT NULL,
	`set`	INTEGER NOT NULL
);
COMMIT;