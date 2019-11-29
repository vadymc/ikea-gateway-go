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
Uses local SQLite DB.
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
create index ix_date_created on stat_data (date_created);

create table quantile_group
(
	id INTEGER
		constraint quantile_group_pk
			primary key autoincrement,
	group_name VARCHAR (40) not null,
	bucket_index INTEGER not null,
	bucket_value INTEGER not null
);
create index ix_group_name_bucket_idx on quantile_group(group_name, bucket_index);
```