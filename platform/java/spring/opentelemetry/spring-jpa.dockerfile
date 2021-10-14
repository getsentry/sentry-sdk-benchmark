FROM maven:3.6.1-jdk-11-slim as maven
WORKDIR /spring
COPY src src
COPY pom.xml pom.xml
RUN mvn package -q
RUN curl https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/v1.6.2/opentelemetry-javaagent-all.jar --output otel.jar
COPY otel.jar otel.jar

FROM openjdk:11.0.3-jdk-slim
WORKDIR /spring
COPY --from=maven /spring/target/hello-spring-1.0-SNAPSHOT.jar app.jar
COPY --from=maven /spring/otel.jar otel.jar

EXPOSE 8080

CMD ["java", "-javaagent:otel.jar", "-server", "-XX:+UseNUMA", "-XX:+UseParallelGC", "-Dotel.traces.exporter=zipkin", "-Dlogging.level.root=OFF", "-jar", "app.jar", "--spring.profiles.active=jpa"]
