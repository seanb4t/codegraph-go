from flask import Flask

app = Flask(__name__)


@app.route("/users/<id>", methods=["GET"])
def get_user(id):
    return None


@app.post("/users")
def create_user():
    return None
