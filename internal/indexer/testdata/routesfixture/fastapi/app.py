from fastapi import FastAPI

app = FastAPI()


@app.get("/items/{id}")
def get_item(id):
    return None
