
Generate a sample user- and repostore with:

```bash
     mkdata.py data.yml
```

After that calling `gin-repod` in the same directory should
start a working server.

Docker:
```bash
     docker build -t gin-repod .
     docker run -it --rm --name gin-repo gin-repod
```
