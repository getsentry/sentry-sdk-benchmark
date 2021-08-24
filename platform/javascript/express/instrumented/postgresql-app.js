/**
 * Module dependencies.
 */
const cluster = require('cluster'),
  numCPUs = require('os').cpus().length,
  express = require('express'),
  helper = require('./helper');

const Sentry = require('@sentry/node');
const Tracing = require("@sentry/tracing");

// Middleware
const bodyParser = require('body-parser');

const Sequelize = require('sequelize');
const sequelize = new Sequelize('hello_world', 'benchmarkdbuser', 'benchmarkdbpass', {
  host: 'tfb-database',
  dialect: 'postgres',
  logging: false,
  pool: {
    min: 20, max: 20
  }
});

const Worlds = sequelize.define('world', {
  id: {
    type: 'Sequelize.INTEGER',
    primaryKey: true
  },
  randomnumber: { type: 'Sequelize.INTEGER' }
}, {
  timestamps: false,
  freezeTableName: true
});

const Fortunes = sequelize.define('fortune', {
  id: {
    type: 'Sequelize.INTEGER',
    primaryKey: true
  },
  message: { type: 'Sequelize.STRING' }
}, {
  timestamps: false,
  freezeTableName: true
});

const randomWorldPromise = () => {
  return Worlds.findOne({
    where: { id: helper.randomizeNum() }
  }).then((results) => {
    return results;
  }).catch((err) => process.exit(1));
};

if (cluster.isMaster) {
  // Fork workers.
  for (let i = 0; i < numCPUs; i++) {
    cluster.fork();
  }

  cluster.on('exit', (worker, code, signal) =>
    console.log('worker ' + worker.pid + ' died'));
} else {
  const app = module.exports = express();

  Sentry.init({
    integrations: [
      // enable HTTP calls tracing
      new Sentry.Integrations.Http({ tracing: true }),
      // enable Express.js middleware tracing
      new Tracing.Integrations.Express({ app }),
    ],
  
    // We recommend adjusting this value in production, or using tracesSampler
    // for finer control
    tracesSampleRate: 1.0,
  });

  // The request handler must be the first middleware on the app
  app.use(Sentry.Handlers.requestHandler());

  // TracingHandler creates a trace for every incoming request
  app.use(Sentry.Handlers.tracingHandler());

  // The error handler must be before any other error middleware and after all controllers
  app.use(Sentry.Handlers.errorHandler());

  app.use(bodyParser.urlencoded({ extended: true }));

  // Set headers for all routes
  app.use((req, res, next) => {
    res.setHeader("Server", "Express");
    return next();
  });

  app.set('view engine', 'pug');
  app.set('views', __dirname + '/views');

  // Routes
  app.get('/db', async (req, res) => {
    let world = await randomWorldPromise();

    res.setHeader("Content-Type", "application/json");
    res.json(world);
  });

  app.get('/queries', async (req, res) => {
    const queries = Math.min(parseInt(req.query.queries) || 1, 500);

    const promisesArray = [];

    for (let i = 0; i < queries; i++) {
      promisesArray.push(randomWorldPromise());
    }

    res.json(await Promise.all(promisesArray))
  });

  app.get('/fortunes', async (req, res) => {
    let fortunes = await Fortunes.findAll();
    const newFortune = { id: 0, message: "Additional fortune added at request time." };
    fortunes.push(newFortune);
    fortunes.sort((a, b) => (a.message < b.message) ? -1 : 1);

    res.render('fortunes/index', { fortunes: fortunes });
  });

  app.get('/update', async (req, res) => {
    const queries = Math.min(parseInt(req.query.queries) || 1, 500);
    const worldPromises = [];

    for (let i = 0; i < queries; i++) {
      worldPromises.push(randomWorldPromise());
    }

    const worldUpdate = (world) => {
      world.randomnumber = helper.randomizeNum();

      return Worlds.update({
            randomnumber: world.randomnumber
          },
          {
            where: { id: world.id }
          }).then((results) => {
        return world;
      }).catch((err) => process.exit(1));
    };

    Promise.all(worldPromises).then((worlds) => {
      const updates = worlds.map((e) => worldUpdate(e));

      Promise.all(updates).then((updated) => {
        res.json(updated);
      });
    });
  });

  app.listen(8080, () => {
    console.log('listening on port 8080');
  });
}
