three.js is three@0.187.0 (MIT, see three.LICENSE), reduced to the classes scene.js imports:

    echo "export { <the names scene.js imports> } from 'three';" > entry.js
    npx esbuild entry.js --bundle --minify --format=esm --outfile=three-0.187.0.js

The version is in the file name, so the server can let browsers keep it for a year. A new build
gets a new name, and scene.js and index.html import it by that name.
