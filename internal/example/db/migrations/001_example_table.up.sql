create table example_table(
    id bigserial,
    foo varchar(256)
);

insert into example_table (foo)
select md5(random()::text)
from generate_series(1, 10000);
