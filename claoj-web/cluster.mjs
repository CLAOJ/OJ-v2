// Runs WEB_CONCURRENCY copies of the stock `next start` in one container.
// A single Next.js process tops out at ~1.5 cores (~11 page renders/s); the
// workers share port 3000 through node:cluster, so nginx still sees one upstream.
import cluster from 'node:cluster';
import os from 'node:os';
import { fileURLToPath } from 'node:url';

const workers = Number(process.env.WEB_CONCURRENCY) || Math.min(4, os.availableParallelism());
let stopping = false;

cluster.setupPrimary({ exec: fileURLToPath(import.meta.resolve('next/dist/bin/next')), args: ['start'] });

for (let i = 0; i < workers; i++) cluster.fork();
console.log(`cluster: started ${workers} next workers`);

cluster.on('exit', (worker, code, signal) => {
    if (stopping) return;
    console.error(`cluster: worker ${worker.process.pid} exited (${signal || code}), restarting`);
    // ponytail: flat 1s back-off; a crash loop restarts forever, add a cap if that ever happens.
    setTimeout(() => cluster.fork(), 1000);
});

for (const sig of ['SIGTERM', 'SIGINT']) {
    process.on(sig, () => {
        stopping = true;
        for (const w of Object.values(cluster.workers)) w.process.kill(sig);
        setTimeout(() => process.exit(0), 10000).unref();
        cluster.on('exit', () => { if (Object.keys(cluster.workers).length === 0) process.exit(0); });
    });
}
