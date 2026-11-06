from fastapi import FastAPI

app = FastAPI()

@app.get("/")
async def root():
    return {"status": "ok", "service": "python-service"}

@app.get("/health")
async def health():
    return {"status": "healthy"}

@app.post("/api")
async def api():
    return {"message": "Hello from Auth service!"}
