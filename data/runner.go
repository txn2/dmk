package data

// A Runner runs a Migration consisting of a
//  - source DB,
//  - source query,
//  - destination DB
//  - destination query.
//  - a collection of transformers
//
// 1) A query is executed on the on source db and foreach result a transformers is invoked along with the
// collection of transformers.
//
// 2) Each source result becomes transformed data (map[string]interface) and is then used as arguments along with
// a query against the destination database.
