FROM registry.access.redhat.com/ubi7/ubi

WORKDIR /opt

ENV JAVA_HOME=/usr/lib/jvm/jre/
ENV HADOOP_HOME=/opt/hadoop-3.1.2
ENV HIVE_HOME=/opt/apache-hive-3.1.2-bin

RUN yum update --disableplugin=subscription-manager -y && rm -rf /var/cache/yum && \
    yum install --disableplugin=subscription-manager java-1.8.0-openjdk-headless -y && \
    yum install --disableplugin=subscription-manager postgresql-devel -y

RUN curl -L https://archive.apache.org/dist/hadoop/core/hadoop-3.1.2/hadoop-3.1.2.tar.gz | tar zxf - && \
    curl -L https://www-us.apache.org/dist/hive/hive-3.1.2/apache-hive-3.1.2-bin.tar.gz | tar zxf -
    
RUN  curl -L https://jdbc.postgresql.org/download/postgresql-42.2.8.jar > ${HIVE_HOME}/lib/postgresql-42.2.8.jar && \
     curl -L https://repo1.maven.org/maven2/org/apache/hadoop/hadoop-aws/3.1.2/hadoop-aws-3.1.2.jar > ${HADOOP_HOME}/lib/hadoop-aws-3.1.2.jar && \
     curl -L https://repo1.maven.org/maven2/com/amazonaws/aws-java-sdk-core/1.11.671/aws-java-sdk-core-1.11.671.jar > ${HADOOP_HOME}/lib/aws-java-sdk-core-1.11.671.jar && \
     curl -L https://repo1.maven.org/maven2/com/amazonaws/aws-java-sdk-s3/1.11.671/aws-java-sdk-s3-1.11.671.jar > ${HADOOP_HOME}/lib/aws-java-sdk-s3-1.11.671.jar && \
     curl -L https://repo1.maven.org/maven2/com/amazonaws/aws-java-sdk-dynamodb/1.11.671/aws-java-sdk-dynamodb-1.11.671.jar >  ${HADOOP_HOME}/lib/aws-java-sdk-dynamodb-1.11.671.jar && \
     cp -v ${HADOOP_HOME}/lib/*aws*.jar ${HIVE_HOME}/lib/
