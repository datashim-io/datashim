DROP TABLE books;
CREATE EXTERNAL TABLE books (id int, title STRING, author STRING, year INT, isbn STRING)
ROW FORMAT DELIMITED FIELDS TERMINATED BY ',' LINES TERMINATED BY '\n'
LOCATION 's3a://book-test/'
tblproperties ("skip.header.line.count"="1");
