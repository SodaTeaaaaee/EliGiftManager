# 第一期内部语义字典和转换器是闭集

模板只能映射到目标模型文档列出的核心语义字段。第一期转换器为 trim、strip_quotes、parseDate、mapEnum、normalizePhone、splitSkuQuantity、joinAddress。缺字段就加进字典，不让模板发明可参与规则、校验、导出或回写的字段。财务类字段不进字典。
