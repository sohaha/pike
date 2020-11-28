///
/// Server配置页
///
import 'dart:html';
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../bloc/bloc.dart';
import '../config/application.dart';
import '../model/config.dart';
import '../widget/button.dart';
import '../widget/error_message.dart';
import '../widget/selector.dart';
import '../widget/table.dart';

@immutable
class ServerPage extends StatefulWidget {
  const ServerPage({
    Key key,
  }) : super(key: key);
  @override
  _ServerPageState createState() => _ServerPageState();
}

class _ServerPageState extends State<ServerPage> {
  final _formKey = GlobalKey<FormState>();

  final _addrCtrl = TextEditingController();
  List<String> _locations;
  String _cache;
  String _compress;
  final _compressMinLengthCtrl = TextEditingController();
  final _compressContentFilterCtrl = TextEditingController();
  final _logFormatCtrl = TextEditingController();
  final _remarkCtrl = TextEditingController();

  String _mode = '';
  final _editMode = 'eidt';
  final _updateMode = 'update';

  ConfigBloc _configBloc;

  @override
  void initState() {
    super.initState();
    _configBloc = BlocProvider.of<ConfigBloc>(context);
  }

  bool get _isEditting => _mode.isNotEmpty;

  bool get _isUpdating => _mode == _updateMode;

  void _reset() {
    _addrCtrl.clear();
    _locations = <String>[];
    _cache = '';
    _compress = '';
    _compressMinLengthCtrl.clear();
    _compressContentFilterCtrl.clear();
    _logFormatCtrl.clear();
    _remarkCtrl.clear();
  }

  // _fillEditor 填充表格内容
  void _fillEditor(ServerConfig element) {
    _addrCtrl.value = TextEditingValue(text: element.addr ?? '');
    _locations = element.locations;
    _cache = element.cache;
    _compress = element.compress;
    _compressMinLengthCtrl.value =
        TextEditingValue(text: element.compressMinLength ?? '');
    _compressContentFilterCtrl.value =
        TextEditingValue(text: element.compressContentTypeFilter ?? '');
    _logFormatCtrl.value = TextEditingValue(text: element.logFormat ?? '');
    _remarkCtrl.value = TextEditingValue(text: element.remark ?? '');
  }

  // _deleteServer 删除服务
  void _deleteServer(ConfigCurrentState state, String addr) {
    final serverList = <ServerConfig>[];
    state.config.servers?.forEach((element) {
      if (element.addr != addr) {
        serverList.add(element);
      }
    });
    _configBloc.add(ConfigUpdate(
        config: state.config.copyWith(
      servers: serverList,
    )));
  }

  // _renderServerList 渲染服务器列表
  Widget _renderServerList(ConfigCurrentState state) {
    // 表格内容
    final contents = state.config.servers
        ?.map((e) => [
              e.addr,
              e.locations,
              e.cache,
              e.compress,
              e.compressMinLength,
              e.compressContentTypeFilter,
              e.logFormat,
              e.remark,
            ])
        ?.toList();
    final doUpdate = (int index) {
      final element = state.config.servers.elementAt(index);
      // 重置当前数据，并将需要更新的配置填充
      _reset();
      _fillEditor(element);

      setState(() {
        _mode = _updateMode;
      });
    };
    final doDelete = (int index) {
      final element = state.config.servers.elementAt(index);
      _deleteServer(state, element.addr);
    };
    return XConfigTable(
      headers: [
        'Addr',
        'Locations',
        'Cache',
        'Compress',
        'Compress Min Length',
        'Compress Content Filter',
        'Log Format',
        'Remark',
      ],
      contents: contents,
      onUpdate: doUpdate,
      onDelete: doDelete,
      columnWidths: <String, double>{
        'Locations': 160,
        'Cache': 140,
        'Compress': 140,
        'Compress Min Length': 200,
        'Compress Content Filter': 220,
      },
    );
  }

  // _addServer 添加server配置，如果已存在，则替换
  void _addServer(ConfigCurrentState state) {
    final addr = _addrCtrl.text?.trim();

    final serverConfig = ServerConfig(
      logFormat: _logFormatCtrl.text?.trim(),
      addr: addr,
      locations: _locations,
      cache: _cache,
      compress: _compress,
      compressMinLength: _compressMinLengthCtrl.text?.trim(),
      compressContentTypeFilter: _compressContentFilterCtrl.text?.trim(),
      remark: _remarkCtrl.text?.trim(),
    );
    final serverList = <ServerConfig>[];
    state.config.servers?.forEach((element) {
      if (element.addr != addr) {
        serverList.add(element);
      }
    });
    serverList.add(serverConfig);
    _configBloc.add(ConfigUpdate(
        config: state.config.copyWith(
      servers: serverList,
    )));
    // 重置当前模式
    setState(() {
      _mode = '';
    });
  }

  // _renderEditor 渲染编辑表单
  Widget _renderEditor(ConfigCurrentState state) {
    if (!_isEditting) {
      return Container();
    }
    final formItems = <Widget>[];
    // 名称
    formItems.add(TextFormField(
      autofocus: true,
      readOnly: _isUpdating,
      controller: _addrCtrl,
      decoration: InputDecoration(
        labelText: 'Addr',
        hintText: 'Please input the addr of server',
      ),
      validator: (v) => v.trim().isNotEmpty ? null : 'addr can not be null',
    ));

    // 选择所使用location
    final locations = state.config.locations?.map((e) => e.name)?.toList();
    formItems.add(Container(
      margin: EdgeInsets.only(
        top: Application.defaultPadding,
      ),
      child: XFormSelector(
        title: 'Locations',
        mutiple: true,
        options: locations,
        values: _locations,
        toggled: true,
        onChanged: (String locations) {
          final result = locations?.split(',');
          setState(() {
            _locations = result;
          });
        },
      ),
    ));

    // 选择所使用缓存
    final caches = state.config.caches?.map((e) => e.name)?.toList();
    formItems.add(Container(
      margin: EdgeInsets.only(
        top: Application.defaultPadding,
      ),
      child: XFormSelector(
        title: 'Cache',
        value: _cache,
        options: caches,
        toggled: true,
        onChanged: (String cache) {
          setState(() {
            _cache = cache;
          });
        },
      ),
    ));

    // 选择所使用压缩配置
    final compresses = state.config.compresses?.map((e) => e.name)?.toList();
    formItems.add(Container(
      margin: EdgeInsets.only(
        top: Application.defaultPadding,
      ),
      child: XFormSelector(
        title: 'Compress',
        value: _compress,
        options: compresses,
        toggled: true,
        onChanged: (String compress) {
          setState(() {
            _compress = compress;
          });
        },
      ),
    ));

    // 设置最小压缩长度
    formItems.add(TextFormField(
      controller: _compressMinLengthCtrl,
      decoration: InputDecoration(
        labelText: 'Compress Min Length',
        hintText: 'Please input the compress min length(1kb, 1mb)',
      ),
      validator: (v) {
        if (v == null || v.isEmpty) {
          return null;
        }
        final reg = RegExp(r'\d+[km]b$');
        if (reg.hasMatch(v)) {
          return null;
        }
        return 'Compress min length is invalid';
      },
    ));

    // 设置压缩内容类型
    formItems.add(TextFormField(
      controller: _compressContentFilterCtrl,
      decoration: InputDecoration(
        labelText: 'Compress Content Filter',
        hintText:
            'Please int the compress content filter, e.g.: text|javascript|json|wasm|xml, optional',
      ),
    ));

    // log format
    formItems.add(TextFormField(
      controller: _logFormatCtrl,
      minLines: 3,
      maxLines: 3,
      decoration: InputDecoration(
        labelText: 'Log Format',
        hintText: 'Please input the log format for server',
      ),
    ));

    // remark
    formItems.add(TextFormField(
      controller: _remarkCtrl,
      minLines: 3,
      maxLines: 3,
      decoration: InputDecoration(
        labelText: 'Remark',
        hintText: 'Please input the remark for server',
      ),
    ));

    return Container(
      margin: EdgeInsets.only(
        top: 3 * Application.defaultPadding,
      ),
      child: Form(
        key: _formKey, //设置globalKey，用于后面获取FormState
        child: Column(
          children: formItems,
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) =>
      BlocBuilder<ConfigBloc, ConfigState>(builder: (context, state) {
        if (state is ConfigErrorState) {
          return XErrorMessage(
            message: state.message,
            title: 'Get server config fail',
          );
        }
        final currentConfig = state as ConfigCurrentState;
        var btnText = _isEditting ? 'Save Server' : 'Add Server';
        if (currentConfig.isProcessing) {
          btnText = 'Processing...';
        }
        return SingleChildScrollView(
          child: Container(
            margin: EdgeInsets.all(3 * Application.defaultPadding),
            child: Column(
              children: [
                _renderServerList(currentConfig),
                _renderEditor(currentConfig),
                XFullButton(
                  margin: EdgeInsets.only(
                    top: 3 * Application.defaultPadding,
                  ),
                  onPressed: () {
                    if (currentConfig.isProcessing) {
                      return;
                    }
                    // 如果是编辑模式，则是添加或更新
                    if (_isEditting) {
                      if (_formKey.currentState.validate()) {
                        _addServer(currentConfig);
                      }
                      return;
                    }
                    // 重置所有数据，设置为编辑模式
                    _reset();
                    setState(() {
                      _mode = _editMode;
                    });
                  },
                  text: Text(btnText),
                ),
              ],
            ),
          ),
        );
      });
}
