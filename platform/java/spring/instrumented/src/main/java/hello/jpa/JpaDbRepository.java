package hello.jpa;

import hello.model.Fortune;
import hello.model.World;
import hello.repository.DbRepository;
import io.sentry.spring.tracing.SentrySpan;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.context.annotation.Profile;
import org.springframework.stereotype.Component;

import java.util.List;

@Component
@Profile("jpa")
public class JpaDbRepository implements DbRepository {
    private final Logger log = LoggerFactory.getLogger(getClass());
    private final WorldRepository worldRepository;
    private final FortuneRepository fortuneRepository;

    public JpaDbRepository(WorldRepository worldRepository, FortuneRepository fortuneRepository) {
        this.worldRepository = worldRepository;
        this.fortuneRepository = fortuneRepository;
    }

    @Override
    @SentrySpan
    public World getWorld(int id) {
        return worldRepository.findById(id).orElse(null);
    }

    @Override
    @SentrySpan
    public World updateWorld(World world, int randomNumber) {
        world.randomnumber = randomNumber;
        return worldRepository.save(world);
    }

    @Override
    public List<Fortune> fortunes() {
        return fortuneRepository.findAll();
    }
}
