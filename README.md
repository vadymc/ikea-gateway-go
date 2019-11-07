# ikea-gateway-go

Gathers statistics about the state of IKEA smart bulbs.

### Required environment variables
```
IKEA_GW_IP
IKEA_GW_PSK
IKEA_DB_PATH
GOOGLE_APPLICATION_CREDENTIALS
```

### DB schema initialization
Uses Firebase storage and local SQLite DB.
```
CREATE TABLE event (
    id           INTEGER      PRIMARY KEY AUTOINCREMENT,
    date_created DATETIME     NOT NULL
                              DEFAULT (CURRENT_TIMESTAMP) 
);

CREATE TABLE stat_data (
    id           INTEGER      PRIMARY KEY AUTOINCREMENT,
    event_id     INTEGER      REFERENCES event (id) NOT NULL,
    group_name   VARCHAR (40) NOT NULL,
    power        BOOLEAN      NOT NULL,
    dimmer       INTEGER      NOT NULL,
    rgb          VARCHAR (7)  NOT NULL,
    date_created DATETIME     NOT NULL
                              DEFAULT (CURRENT_TIMESTAMP) 
);
```