-- name: ExampleQuery :many
select * from example_table
where foo < $1;

-- name: ExampleQuery2 :many
select * from example_table
where foo = $1;