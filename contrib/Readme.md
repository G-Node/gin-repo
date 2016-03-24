
Generate keys with:
```bash
     ssh-keygen -t rsa -b 4096 -C "blue" -f blue.rsa -P ""
```

Docker:
```bash
     docker build -t gin-repod .
     docker run -it --rm --name gin-repo gin-repod
```