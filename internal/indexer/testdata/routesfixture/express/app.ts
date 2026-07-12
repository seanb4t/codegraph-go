import express from "express";

function getUserHandler() {}

function createUserHandler() {}

const app = express();
app.get("/users/:id", getUserHandler);
app.post("/users", createUserHandler);
