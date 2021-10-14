FROM maven:3.6.1-jdk-11-slim as maven
WORKDIR /spring
COPY src src
COPY pom.xml pom.xml
RUN mvn package -q

FROM openjdk:11.0.3-jdk-slim
WORKDIR /spring
COPY --from=maven /spring/target/hello-spring-1.0-SNAPSHOT.jar app.jar
COPY otel.jar otel.jar

EXPOSE 8080

CMD ["java", "-javaagent:otel.jar", "-server", "-XX:+UseNUMA", "-XX:+UseParallelGC", "-Dotel.traces.exporter=zipkin", "-jar", "app.jar", "--spring.profiles.active=jpa"]
