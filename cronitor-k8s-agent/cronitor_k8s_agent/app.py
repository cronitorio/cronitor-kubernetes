from flask import Flask

app = Flask(__name__)


@app.route('/healthz')
def healthz():
    return {
        "status": "OK",
    }


